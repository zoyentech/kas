package balance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/zoyentech/kas/internal/domain"
)

// JevSource 经 Jev 余额端点查询（Story 8.2.2.1；O-2 协议未定——最小契约
// GET {endpoint} + Bearer，响应 {"amount":x,"unit":"credit"}；协议文档到位
// 仅改本文件）。错误只含 env 名与状态码，不含 key 与余额值。
type JevSource struct {
	endpoint string
	apiKey   string
	http     *http.Client
}

// NewJevSource 由 env 名构建（endpointEnv/apiKeyEnv 未设置或为空 → 明确错误）。
func NewJevSource(endpointEnv, apiKeyEnv string) (*JevSource, error) {
	endpoint := os.Getenv(endpointEnv)
	if endpoint == "" {
		return nil, fmt.Errorf("%w: decision.jev.balance_endpoint_env: 环境变量 %s 未设置或为空",
			domain.ErrInvalidConfig, endpointEnv)
	}
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: decision.jev.%s: 环境变量 %s 未设置或为空",
			domain.ErrInvalidConfig, "api_key_env", apiKeyEnv)
	}
	return &JevSource{endpoint: endpoint, apiKey: apiKey, http: &http.Client{Timeout: 2 * time.Second}}, nil
}

// jevBalanceResponse 余额响应最小契约（严格解析：未知字段拒绝）。
type jevBalanceResponse struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// Balance 执行查询（单次、2s 独立超时、错误脱敏）。
func (s *JevSource) Balance(ctx context.Context) (Info, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		return Info{}, fmt.Errorf("kas/balance: 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.http.Do(req)
	if err != nil {
		return Info{}, fmt.Errorf("kas/balance: 查询失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("kas/balance: 上游状态码 %d", resp.StatusCode)
	}
	var br jevBalanceResponse
	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&br); err != nil {
		return Info{}, fmt.Errorf("%w: 余额响应解析失败（畸形体或未知字段）", domain.ErrProviderContract)
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return Info{}, fmt.Errorf("%w: 余额响应含多个 JSON 文档", domain.ErrProviderContract)
	}
	return Info{Source: "jev", Amount: br.Amount, Unit: br.Unit, FetchedAt: time.Now()}, nil
}
