package kas

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/kas-runtime/kas/internal/cache"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/obs"
	"github.com/kas-runtime/kas/internal/orchestrator"
	"github.com/kas-runtime/kas/internal/runtime"
)

// Dependencies 是宿主/装配层注入的运行依赖。EvidenceProvider 必需；
// DecisionProvider 可选（缺失时降级链落到确定性规则，S5）；
// Cache 可选（缺省且配置启用缓存时用内置 MemCache）；
// PermissionChecker 可选（O-7：nil = 不做权限过滤，生产部署应注入宿主
// 判定函数或声明式映射）；HealthChecks/TraceSink 可选（S6）。
// 宿主传入且共享的资源（HTTP client、registry 等）归宿主管理。
type Dependencies struct {
	EvidenceProvider  EvidenceProvider
	DecisionProvider  DecisionProvider
	Cache             EvidenceCache
	PermissionChecker PermissionChecker
	HealthChecks      HealthChecks
	TraceSink         TraceSink
}

// Option 定制 New 的装配行为（S6 增加 metrics Registerer、TraceSink 等）。
type Option func(*options)

type options struct{}

// Client 是 KAS SDK 入口。方法并发安全；请求间无会话持久状态（F-012）。
type Client struct {
	manager *runtime.Manager
	engine  *orchestrator.Engine
	bus     *obs.Bus
	closed  atomic.Bool
}

// New 同步校验配置与必要依赖后装配 Client：构建初始 Runtime Snapshot 并
// 加载为 current（docs/design/03 §1）。密钥由宿主 secret manager/env 注入，
// 不写入 YAML 或错误文本。
func New(cfg Config, deps Dependencies, opts ...Option) (*Client, error) {
	o := options{}
	for _, apply := range opts {
		apply(&o)
	}
	if deps.EvidenceProvider == nil {
		return nil, fmt.Errorf("%w: Dependencies.EvidenceProvider: 必需依赖不能为 nil（必需组件见 health.required_components）", ErrInvalidConfig)
	}
	snap, err := runtime.BuildSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	bus := obs.NewBus()
	mgr := runtime.NewManager(snap)
	engine := orchestrator.NewEngine(mgr, bus, deps.EvidenceProvider, deps.DecisionProvider)

	// Evidence 缓存：宿主未注入且配置启用时使用内置有界内存缓存
	engine.Cache = deps.Cache
	if engine.Cache == nil && cfg.Evidence.Cache.Enabled {
		engine.Cache = cache.NewMemCache(
			cfg.Evidence.Cache.MaxEntries,
			cfg.Evidence.Cache.MaxBytes,
			cfg.Evidence.Cache.MaxItemBytes,
			bus,
		)
	}
	engine.Perm = deps.PermissionChecker
	return &Client{manager: mgr, engine: engine, bus: bus}, nil
}

// Answer 执行一次问答编排（初判→检索→决策→补检→组装）。请求开始读取
// 一次 snapshot 并全程持有（F-011/F-012）。
func (c *Client) Answer(ctx context.Context, req Request) (Response, error) {
	if c.closed.Load() {
		return Response{}, ErrClosed
	}
	resp, err := c.engine.Answer(ctx, req)
	if err != nil {
		return resp, orchestrator.MapToPublicError(err)
	}
	return resp, nil
}

// Health 聚合组件健康；使用短独立 timeout，不执行用户查询或模型生成。
func (c *Client) Health(ctx context.Context) HealthReport {
	if c.closed.Load() {
		return HealthReport{Status: domain.StatusNotReady, Components: map[string]ComponentHealth{
			"client": {Status: domain.ComponentNotReady, Detail: "client closed"},
		}}
	}
	return HealthReport{Status: domain.StatusHealthy, CheckedAt: orchestrator.SystemClock{}.Now()} // TODO(S6): Story 6.1.1.1 聚合实现
}

// Reload 依次解析、Validate、准备候选 runtime、计算脱敏 config_version
// 后原子替换；任一步失败保持原 snapshot（F-011）。成功返回新 config_version。
func (c *Client) Reload(ctx context.Context, r io.Reader) (string, error) {
	if c.closed.Load() {
		return "", ErrClosed
	}
	res, err := runtime.Reload(ctx, c.manager, r)
	if err != nil {
		return "", err
	}
	return res.Version, nil
}

// ConfigVersion 返回当前运行快照的 config_version（诊断用途）。
func (c *Client) ConfigVersion() string { return c.manager.Version() }

// Close 停止新请求并关闭 SDK 自有资源；幂等。在途请求自然完成或由其
// ctx 取消。宿主传入且共享的资源归宿主管理（docs/design/03 §6）。
func (c *Client) Close() error {
	c.closed.Store(true)
	return nil
}
