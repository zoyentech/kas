// Package obs 实现可观测性：Health 聚合（healthy/degraded/not_ready，短独立
// 超时）、Prometheus collector（低基数 label allowlist、宿主 Registerer 注入）
// 与 TraceSink（include 白名单、脱敏规则、TraceID 透传）。
// 设计依据：docs/design/04 §4~§6。
package obs
