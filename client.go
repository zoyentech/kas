package kas

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/zoyentech/kas/internal/balance"
	balanceadapters "github.com/zoyentech/kas/internal/balance/adapters"
	"github.com/zoyentech/kas/internal/cache"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/decision/dcache"
	"github.com/zoyentech/kas/internal/decision/fallback"
	"github.com/zoyentech/kas/internal/decision/jev"
	"github.com/zoyentech/kas/internal/decision/rules"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/obs/health"
	"github.com/zoyentech/kas/internal/obs/metrics"
	"github.com/zoyentech/kas/internal/obs/trace"
	"github.com/zoyentech/kas/internal/orchestrator"
	"github.com/zoyentech/kas/internal/runtime"
)

// Dependencies 是宿主/装配层注入的运行依赖。EvidenceProvider 必需；
// DecisionProvider 可选（缺失时按配置装配降级链）；Cache 可选（缺省且
// 配置启用时用内置 MemCache）；PermissionChecker 可选（O-7）；
// HealthChecks/TraceSink 可选。宿主传入且共享的资源（HTTP client、
// registry 等）归宿主管理。
type Dependencies struct {
	EvidenceProvider  EvidenceProvider
	DecisionProvider  DecisionProvider
	Cache             EvidenceCache
	PermissionChecker PermissionChecker
	HealthChecks      HealthChecks
	TraceSink         TraceSink
	// BalanceSource 可选（v1.3 F-016）：注入自定义计费源；nil 且配置
	// balance.enabled 时按 decision.jev.balance_endpoint_env 构建 Jev 源。
	BalanceSource BalanceSource
}

// Option 定制 New 的装配行为。
type Option func(*options)

type options struct {
	registerer prometheus.Registerer
}

// WithPrometheusRegisterer 注入宿主 Prometheus Registerer（F-006；D-3）。
// KAS 注册自有 collector、不开 /metrics listener；Close 时注销。
func WithPrometheusRegisterer(reg prometheus.Registerer) Option {
	return func(o *options) { o.registerer = reg }
}

// Client 是 KAS SDK 入口。方法并发安全；请求间无会话持久状态（F-012）。
type Client struct {
	manager   *runtime.Manager
	engine    *orchestrator.Engine
	bus       *obs.Bus
	deps      Dependencies
	collector *metrics.Collector
	// balanceCfg 保存当前余额配置（SetBalanceConfig 热更新；balanceCfgMu 保护）。
	balanceCfgMu sync.Mutex
	// balanceMgr 可空（余额监测禁用时为 nil；v1.4 F-016 多源）。
	balanceMgr *balance.Manager
	balanceCfg config.BalanceConfig
	closed     atomic.Bool
	closeOnce  sync.Once
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

	// 决策链（F-007/F-008）：宿主注入的 DecisionProvider 作为首选级
	// （Jev 角色）；未注入时按配置构建 Jev 客户端。降级链与护栏始终生效。
	chain, err := buildDecisionChain(cfg, bus, deps.DecisionProvider)
	if err != nil {
		return nil, err
	}
	engine.Decision = chain

	c := &Client{manager: mgr, engine: engine, bus: bus, deps: deps}

	// 可观测接线（S6）：metrics（宿主 Registerer）+ trace（白名单脱敏）
	if cfg.Metrics.Enabled {
		col := metrics.New(cfg.Metrics)
		if err := col.RegisterTo(o.registerer); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
		bus.Subscribe(col)
		c.collector = col
	}
	if cfg.Tracing.Enabled {
		sink := deps.TraceSink
		if sink == nil {
			sink = trace.NoopSink{}
		}
		bus.Subscribe(trace.NewAdapter(sink, trace.NewRedactor(cfg.Tracing.Include)))
	}

	// 余额监测（v1.4 F-016 多源）：内置源按 env 存在自动注册（jev/
	// deepseek/aliyun），宿主注入源注册为 "host"；至少一个源可用否则 fail-fast。
	if cfg.Balance.Enabled {
		mgr := balance.NewManager(bus)
		c.balanceMgr = mgr
		c.balanceCfg = cfg.Balance

		registered := 0
		// Jev（可选端点，O-2 最小契约）
		if cfg.Decision.Provider == "jev" && cfg.Decision.Jev.BalanceEndpointEnv != "" {
			if js, err := balance.NewJevSource(cfg.Decision.Jev.BalanceEndpointEnv, cfg.Decision.Jev.APIKeyEnv); err == nil {
				mgr.Upsert("jev", js, cfg.Balance)
				registered++
			}
		}
		// DeepSeek（DEEPSEEK_TOKEN；端点可经 DEEPSEEK_BALANCE_ENDPOINT 覆盖）
		if ds, err := balanceadapters.NewDeepSeek("DEEPSEEK_TOKEN", "DEEPSEEK_BALANCE_ENDPOINT"); err == nil {
			mgr.Upsert("deepseek", ds, cfg.Balance)
			registered++
		}
		// 阿里云账户现金余额（BSS 只读凭据）
		aliyunSrc, aliyunErr := balanceadapters.NewAliyunBSS("ALIBABA_CLOUD_ACCESS_KEY_ID",
			"ALIBABA_CLOUD_ACCESS_KEY_SECRET", "ALIBABA_CLOUD_SECURITY_TOKEN",
			"ALIBABA_CLOUD_BSS_ENDPOINT")
		if aliyunErr == nil {
			mgr.Upsert("aliyun", aliyunSrc, cfg.Balance)
			registered++
			// qwen（DashScope/Qwen 模型计费=阿里云账户现金余额；QWEN_APIKEY
			// 标识在用 qwen，重标为组件视图分列展示；阈值独立可配）
			if os.Getenv("QWEN_APIKEY") != "" {
				mgr.Upsert("qwen", balance.WithName("qwen", aliyunSrc), cfg.Balance)
				registered++
			}
		}
		// 宿主注入源（自定义计费源）
		if deps.BalanceSource != nil {
			mgr.Upsert("host", deps.BalanceSource, cfg.Balance)
			registered++
		}
		if registered == 0 {
			_ = c.Close()
			return nil, fmt.Errorf("%w: balance.enabled=true 但无可用余额源（需 decision.jev.balance_endpoint_env / DEEPSEEK_TOKEN / ALIBABA_CLOUD_ACCESS_KEY_ID 任一，或注入 Dependencies.BalanceSource）", ErrInvalidConfig)
		}
	}
	return c, nil
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

// Health 聚合组件健康（F-005）：并发探测、每组件短独立 check_timeout、
// 不执行用户查询或模型生成、不访问 Evidence cache。探测频率归宿主控制。
func (c *Client) Health(ctx context.Context) HealthReport {
	if c.closed.Load() {
		return HealthReport{Status: domain.StatusNotReady, Components: map[string]ComponentHealth{
			"client": {Status: domain.ComponentNotReady, Detail: "client closed"},
		}, CheckedAt: orchestrator.SystemClock{}.Now()}
	}
	snap := c.manager.Current()
	if snap == nil {
		return HealthReport{Status: domain.StatusNotReady, CheckedAt: orchestrator.SystemClock{}.Now()}
	}
	report := health.Probe(ctx, snap.Config().Health, health.Deps{
		Evidence: c.deps.EvidenceProvider,
		Decision: c.engine.Decision,
		Extra:    c.deps.HealthChecks,
	}, c.bus)
	// 余额信息组件（不参与 required/optional 聚合，仅展示；v1.4 按源分列
	// "balance:<source>"——管理页与仪表板据此区分组件余额）。
	if c.balanceMgr != nil {
		if report.Components == nil {
			report.Components = map[string]ComponentHealth{}
		}
		for name, snap := range c.balanceMgr.SnapshotAll() {
			st := domain.ComponentDegraded
			switch snap.State {
			case balance.StateNormal:
				st = domain.ComponentHealthy
			case balance.StateCritical, balance.StateUnknown:
				st = domain.ComponentNotReady
			}
			report.Components["balance:"+name] = ComponentHealth{
				Status: st,
				Detail: fmt.Sprintf("amount=%.2f%s（state=%s since=%s）",
					snap.Last.Amount, snap.Last.Unit, snap.State, snap.StateAt.Format(time.RFC3339)),
			}
		}
	}
	return report
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

// Balance 返回默认源（优先 jev，否则字典序首个）的余额快照；未启用余额监测
// 时 state 恒为 unknown。unknown 态下 last 为最近一次成功值（失联期间保留展示）。
func (c *Client) Balance() (BalanceInfo, BalanceState) {
	if c.balanceMgr == nil {
		return BalanceInfo{}, BalanceUnknown
	}
	name, ok := c.balanceMgr.DefaultName()
	if !ok {
		return BalanceInfo{}, BalanceUnknown
	}
	info, state, _, _ := c.balanceMgr.Snapshot(name)
	return BalanceInfo(info), BalanceState(state)
}

// BalanceOf 返回指定源余额快照（多源；v1.4）。
func (c *Client) BalanceOf(name string) (BalanceInfo, BalanceState, bool) {
	if c.balanceMgr == nil {
		return BalanceInfo{}, BalanceUnknown, false
	}
	info, state, _, ok := c.balanceMgr.Snapshot(name)
	return BalanceInfo(info), BalanceState(state), ok
}

// BalanceSources 返回已注册余额源名（排序稳定）。
func (c *Client) BalanceSources() []string {
	if c.balanceMgr == nil {
		return nil
	}
	return c.balanceMgr.Names()
}

// UpsertBalanceSource 运行时注册/替换余额源（热更新：宿主管理后台更换组件
// 凭据或新增计费源时调用，无需重建 Client）。global 为当前全局默认配置
// （通常传 LoadConfig 的 balance 节；源级覆盖经 balance.sources[name] 生效）。
func (c *Client) UpsertBalanceSource(name string, src BalanceSource, global BalanceConfig) error {
	if c.closed.Load() {
		return ErrClosed
	}
	if c.balanceMgr == nil {
		return fmt.Errorf("%w: 余额监测未启用（balance.enabled=false）", ErrInvalidConfig)
	}
	if src == nil || name == "" {
		return fmt.Errorf("%w: 余额源 name/src 不能为空", ErrInvalidRequest)
	}
	c.balanceMgr.Upsert(name, src, config.BalanceConfig(global))
	return nil
}

// RemoveBalanceSource 运行时移除余额源（停止其轮询）。
func (c *Client) RemoveBalanceSource(name string) {
	if c.balanceMgr != nil {
		c.balanceMgr.Remove(name)
	}
}

// SetBalanceConfig 运行时热更新余额配置（阈值/周期；对全部已注册源生效，
// 源级覆盖经 Sources[name] 生效）。典型来源：宿主 Reload 新配置后同步。
func (c *Client) SetBalanceConfig(global BalanceConfig) {
	c.balanceCfgMu.Lock()
	c.balanceCfg = global
	c.balanceCfgMu.Unlock()
	if c.balanceMgr != nil {
		for _, name := range c.balanceMgr.Names() {
			c.balanceMgr.SetConfig(name, config.BalanceConfig(global))
		}
	}
}

// ConfigVersion 返回当前运行快照的 config_version（诊断用途）。
func (c *Client) ConfigVersion() string { return c.manager.Version() }

// Close 停止新请求并关闭 SDK 自有资源；幂等。注销自有 metrics collector；
// 在途请求自然完成或由其 ctx 取消。宿主传入且共享的资源归宿主管理
// （docs/design/03 §6）。
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		if c.collector != nil {
			c.collector.Unregister()
		}
		if c.balanceMgr != nil {
			c.balanceMgr.Close() // 停止全部余额轮询 goroutine（goleak 验证）
		}
	})
	return nil
}

// jevBreakerConfigOf 读取熔断配置（nil → 默认开启 {5, 30s}）。
func jevBreakerConfigOf(cfg Config) jev.BreakerConfig {
	if cfg.Decision.Jev.Breaker == nil {
		return jev.DefaultBreakerConfig()
	}
	br := cfg.Decision.Jev.Breaker
	out := jev.BreakerConfig{
		Enabled:          br.Enabled,
		FailureThreshold: br.FailureThreshold, Cooldown: br.Cooldown.Std(),
	}
	if out.Enabled {
		if out.FailureThreshold <= 0 {
			out.FailureThreshold = 5
		}
		if out.Cooldown <= 0 {
			out.Cooldown = 30 * time.Second
		}
	}
	return out
}

// buildDecisionChain 装配降级链（Story 5.3.1.1 / 5.3.2.1 T3）；
// primary 非空时作为 Jev 角色首选级（宿主自定义决策器同样获得降级保护）。
func buildDecisionChain(cfg Config, bus *obs.Bus, primary domain.DecisionProvider) (domain.DecisionProvider, error) {
	guard, err := rules.New(cfg.Decision.Fallback.Rules)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	rulesProv, err := rules.NewProvider(cfg.Decision.Fallback.Rules)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	var stages []fallback.Stage
	if primary != nil {
		stages = append(stages, fallback.Stage{Route: fallback.RouteJev, Provider: primary})
	} else if cfg.Decision.Provider == "jev" {
		jc, err := jev.LoadConfig(cfg.Decision.Jev, cfg.Timeouts.Rejudge.Std())
		if err != nil {
			return nil, err
		}
		// 熔断（v1.3 F-019）：默认开启；配置可关/调参（Story 8.5.2.1）
		bcfg := jevBreakerConfigOf(cfg)
		stages = append(stages, fallback.Stage{
			Route:    fallback.RouteJev,
			Provider: jev.New(jc).WithBreaker(bcfg, bus),
		})
	}
	stages = append(stages, fallback.Stage{Route: fallback.RouteRules, Provider: rulesProv})
	var dcacheProv *dcache.Provider
	if cfg.Decision.Fallback.Cache.Enabled {
		dcacheProv = dcache.NewProvider(
			cfg.Decision.Fallback.Cache.MaxEntries,
			cfg.Decision.Fallback.Cache.TTL.Std(),
		)
		stages = append(stages, fallback.Stage{Route: fallback.RouteCache, Provider: dcacheProv})
	}
	chain := fallback.New(stages, bus, guard)
	if dcacheProv != nil {
		chain.CacheWriter = dcacheProv // Jev 成功决策回填（正向决策；禁写终态）
	}
	return chain, nil
}
