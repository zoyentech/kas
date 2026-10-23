package kas

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/kas-runtime/kas/internal/runtime"
)

// Dependencies 是宿主/装配层注入的运行依赖。EvidenceProvider 必需；
// DecisionProvider 可选（缺失时降级链落到确定性规则）；Cache/HealthChecks/
// TraceSink 可选（缺省用 KAS 默认实现或空实现）。
type Dependencies struct {
	EvidenceProvider EvidenceProvider
	DecisionProvider DecisionProvider
	Cache            EvidenceCache
	HealthChecks     HealthChecks
	TraceSink        TraceSink
}

// Option 定制 New 的装配行为。
type Option func(*options)

type options struct{}

// Client 是 KAS SDK 入口。方法并发安全；请求间无会话持久状态（F-012）。
type Client struct {
	manager *runtime.Manager
	deps    Dependencies
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
	var vs []string
	if deps.EvidenceProvider == nil {
		vs = append(vs, "Dependencies.EvidenceProvider: 必需依赖不能为 nil（必需组件见 health.required_components）")
	}
	if len(vs) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, vs)
	}
	snap, err := runtime.BuildSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{manager: runtime.NewManager(snap), deps: deps}, nil
}

// Answer 执行一次问答编排（初判→检索→决策→补检→组装）。
func (c *Client) Answer(ctx context.Context, req Request) (Response, error) {
	if c.closed.Load() {
		return Response{}, ErrClosed
	}
	return Response{}, errNotImplemented // TODO(S3): Story 3.3.1.1 编排流水线
}

// Health 聚合组件健康；使用短独立 timeout，不执行用户查询或模型生成。
func (c *Client) Health(ctx context.Context) HealthReport {
	return HealthReport{} // TODO(S6): Story 6.1.1.1 聚合实现
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

// Close 停止新请求并关闭 SDK 自有资源；宿主传入且共享的资源归宿主管理。
func (c *Client) Close() error {
	c.closed.Store(true)
	return nil
}
