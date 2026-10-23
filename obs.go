package kas

import "context"

// TODO(S6): Epic 6.1/6.3 落地；此处为 docs/design/03 §5/§6 占位。

// ComponentHealth 是组件健康状态（供 EvidenceProvider/DecisionProvider 报告）。
type ComponentHealth struct{} // TODO(S6): Story 6.1.1.1 细化（status/detail）

// HealthReport 是 Client.Health 的聚合结果：总状态为
// healthy | degraded | not_ready（F-005）。
type HealthReport struct{} // TODO(S6): Story 6.1.1.1 细化

// HealthChecks 是宿主注入的组件健康检查集合。
type HealthChecks struct{} // TODO(S6): Story 6.1.1.1 细化

// TraceEvent 是一次请求内的事件/字段记录，遵循 include 白名单与脱敏规则
// （F-009：config_version、provider、model、model_version、fallback_route）。
type TraceEvent struct{} // TODO(S6): Story 6.3.1.1 细化

// TraceSink 由宿主注入；KAS 不落盘、不外发 trace。
type TraceSink interface {
	Record(ctx context.Context, event TraceEvent)
}
