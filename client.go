package kas

import (
	"context"
	"io"
)

// TODO(S3): Story 3.3.2.1 落地 Client 并发安全生命周期；此处为
// docs/design/03 §1/§5 占位。

// Dependencies 是宿主/装配层注入的运行依赖。
type Dependencies struct {
	EvidenceProvider EvidenceProvider
	DecisionProvider DecisionProvider
	Cache            EvidenceCache
	HealthChecks     HealthChecks
	TraceSink        TraceSink
}

// Option 定制 New 的装配行为（如 Prometheus Registerer、时钟注入）。
type Option func(*options) // TODO(S3): Story 3.3.2.1 细化

type options struct{}

// Client 是 KAS SDK 入口。方法并发安全；无会话持久状态（F-012）。
type Client struct{}

// New 同步校验预算等式、duration 溢出、必要依赖、cache 容量、有效期策略
// 与 provider 参数；密钥由宿主 secret manager/env 注入，不写入 YAML 或
// 错误文本。
func New(cfg Config, deps Dependencies, opts ...Option) (*Client, error) {
	return newClient(cfg, deps, opts...)
}

// Answer 执行一次问答编排（初判→检索→决策→补检→组装）。
func (c *Client) Answer(ctx context.Context, req Request) (Response, error) {
	return c.answer(ctx, req)
}

// Health 聚合组件健康；使用短独立 timeout，不执行用户查询或模型生成。
func (c *Client) Health(ctx context.Context) HealthReport {
	return c.health(ctx)
}

// Reload 依次解析、Validate、准备候选 runtime/provider、计算脱敏
// config_version 后原子替换；任一步失败保持原 snapshot（F-011）。
func (c *Client) Reload(ctx context.Context, r io.Reader) (string, error) {
	return c.reload(ctx, r)
}

// Close 停止新请求并关闭 SDK 自有资源；宿主传入且共享的资源归宿主管理。
func (c *Client) Close() error {
	return c.close()
}
