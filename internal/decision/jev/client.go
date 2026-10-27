// Package jev 封装 Jev 私有协议 adapter（F-007：协议只存在于本包；
// O-2 协议文档未到位，按 docs/design/03 §3 最小契约实现，文档到位后
// 仅改本包）。密钥经环境变量注入，绝不入日志/错误文本。
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
)

// Config 是运行所需的环境注入配置（Story 5.2.1.1 T1）。
type Config struct {
	Endpoint     string
	APIKey       string // 仅存内存；Stringer/redaction 防打印
	Model        string
	ModelVersion string
	MaxAttempts  int
	Timeout      time.Duration // 单次调用上限（默认 rejudge 预算；实际仍受阶段 ctx 约束）
}

func (c Config) String() string {
	return fmt.Sprintf("jev.Config{Endpoint: %q, APIKey: <redacted>}", c.Endpoint)
}

// LoadConfig 从环境变量读取（env 名来自配置 *_env 字段）。
// 缺失/为空 → ErrInvalidConfig（字段路径指向 env 名，绝不回显值）
// （docs/05 矩阵选构造期报错）。
func LoadConfig(cfg config.JevConfig, defaultTimeout time.Duration) (Config, error) {
	out := Config{Model: cfg.Model, ModelVersion: cfg.ModelVersion, MaxAttempts: cfg.MaxAttempts}
	if cfg.MaxAttempts <= 0 {
		out.MaxAttempts = 1 // docs/03 §3：默认单次
	}
	out.Timeout = defaultTimeout
	if cfg.Timeout != nil {
		out.Timeout = cfg.Timeout.Std()
	}

	endpoint, err := requireEnv(cfg.EndpointEnv)
	if err != nil {
		return Config{}, fmt.Errorf("%w: decision.jev.%s: %v", domain.ErrInvalidConfig, "endpoint_env", err)
	}
	out.Endpoint = endpoint

	apiKey, err := requireEnv(cfg.APIKeyEnv)
	if err != nil {
		return Config{}, fmt.Errorf("%w: decision.jev.%s: %v", domain.ErrInvalidConfig, "api_key_env", err)
	}
	out.APIKey = apiKey
	return out, nil
}

func requireEnv(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("环境变量 %s 未设置或为空", name)
	}
	return v, nil
}

// jevRequest 是最小契约请求体（O-2 待定；候选组必要字段全集）。
type jevRequest struct {
	Question      string     `json:"question"`
	Groups        []jevGroup `json:"candidate_groups"`
	ValidAt       time.Time  `json:"valid_at"`
	ConfigVersion string     `json:"config_version"`
	ModelHint     string     `json:"model_hint,omitempty"`
}

type jevGroup struct {
	ID          string   `json:"id"`
	EvidenceIDs []string `json:"evidence_ids"`
	Score       float64  `json:"score"`
}

// jevResponse 是最小契约响应体（严格解析：未知字段拒绝）。
type jevResponse struct {
	Action         string   `json:"action"`
	TargetGroupIDs []string `json:"target_group_ids"`
	NeedSupplement bool     `json:"need_supplement"`
	ReasonCode     string   `json:"reason_code"`
	Confidence     float64  `json:"confidence"`
	Model          string   `json:"model"`
	ModelVersion   string   `json:"model_version"`
	// AnswerText 上游生成的回答草稿（O-2 待定字段；KAS 只组装与校验）。
	AnswerText string `json:"answer_text"`
}

// Client 是 Jev HTTP 客户端，实现 domain.DecisionProvider。
type Client struct {
	cfg  Config
	http *http.Client
	// brk 可选熔断器（v1.3 F-019；nil = 不熔断）。
	brk *Breaker
}

// New 构建 Jev 客户端。
func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			// transport 上限 = 配置 timeout；阶段 ctx 再收紧（docs/04 §1）
			Timeout: cfg.Timeout,
		},
	}
}

// WithBreaker 挂载熔断器（链式；bus 用于 BreakerEvent 分发）。
func (c *Client) WithBreaker(cfg BreakerConfig, bus *obs.Bus) *Client {
	if !cfg.Enabled {
		return c
	}
	c.brk = NewBreaker(cfg, bus)
	return c
}

// BreakerState 返回熔断状态（Health detail 消费；未挂载恒 closed）。
func (c *Client) BreakerState() string {
	if c.brk == nil {
		return BreakerClosed.String()
	}
	return c.brk.State().String()
}

// Decide 执行单次调用（Story 5.2.1.1 T3：拒绝重试，即使配置 >1）；
// 响应经严格解析 + 统一校验器（Story 5.2.1.2）；模型版本捕获进 Decision。
// 熔断 open 期快速失败（不烧 rejudge 预算，链直接降级；Story 8.5.2.1）。
func (c *Client) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	if c.brk != nil {
		if err := c.brk.Allow(); err != nil {
			return domain.Decision{}, err
		}
	}
	d, err := c.decideOnce(ctx, in)
	if c.brk != nil {
		c.brk.Record(err == nil)
	}
	return d, err
}

// decideOnce 单次 HTTP 调用（原 Decide 主体）。
func (c *Client) decideOnce(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	if c.cfg.MaxAttempts != 1 {
		// docs/03 §3：Jev 默认最多调用一次，以免重试耗尽补检/余量预算。
		// 配置层已限制可装入预算的重试；此处仍按单次执行语义。
		_ = c.cfg.MaxAttempts
	}
	body, err := encodeRequest(in)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("kas/jev: 请求编码失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return domain.Decision{}, fmt.Errorf("kas/jev: 构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return domain.Decision{}, fmt.Errorf("kas/jev: 调用失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return domain.Decision{}, fmt.Errorf("kas/jev: 上游状态码 %d", resp.StatusCode)
	}
	jr, err := parseResponse(resp.Body)
	if err != nil {
		return domain.Decision{}, err
	}
	d := domain.Decision{
		Action:         domain.ActionType(jr.Action),
		TargetGroupIDs: jr.TargetGroupIDs,
		NeedSupplement: jr.NeedSupplement,
		ReasonCode:     jr.ReasonCode,
		Confidence:     jr.Confidence,
		Provider:       "jev",
		// 模型版本捕获（Story 5.2.1.2 T2）：响应优先，回退配置声明；
		// 值仅进 Decision/trace，不进 metric label。
		Model:        jr.Model,
		ModelVersion: jr.ModelVersion,
		AnswerText:   jr.AnswerText,
	}
	if d.Model == "" {
		d.Model = c.cfg.Model
	}
	if d.ModelVersion == "" {
		d.ModelVersion = c.cfg.ModelVersion
	}
	// 统一响应校验器（与任何 provider 相同的信任边界）
	if err := validateAgainst(in, d); err != nil {
		return domain.Decision{}, err
	}
	return d, nil
}

// Health 构造期已校验 env；无独立健康端点（O-2 待定），恒报 healthy；
// detail 携带熔断状态（v1.3：运维可见性）。
func (c *Client) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{
		Status: domain.ComponentHealthy,
		Detail: "jev env 已注入；无独立健康端点（O-2 待定）；breaker=" + c.BreakerState(),
	}
}

// encodeRequest 映射 DecisionInput → jevRequest（Story 5.2.1.1 T2 映射单测对象）。
func encodeRequest(in domain.DecisionInput) ([]byte, error) {
	req := jevRequest{
		Question:      in.Question,
		ValidAt:       in.ValidAt,
		ConfigVersion: in.ConfigVersion,
		ModelHint:     in.ModelHint,
	}
	for _, g := range in.Groups {
		ids := make([]string, 0, len(g.Evidence))
		for _, e := range g.Evidence {
			ids = append(ids, e.EvidenceID)
		}
		req.Groups = append(req.Groups, jevGroup{ID: g.ID, EvidenceIDs: ids, Score: g.Score})
	}
	return json.Marshal(req)
}

// validateAgainst 复用统一校验器（internal/decision.ValidateDecision 的
// 本包薄引用，保持包边界）。
func validateAgainst(in domain.DecisionInput, d domain.Decision) error {
	return decisionValidate(in, d)
}
