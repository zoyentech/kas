package obs

import "time"

// Event 是内部事件总线的事件 sealed 接口（Story 3.3.1.2 T1）。
// 事件字段 ↔ docs/design/04 §5 指标表、§6 trace 字段的映射：
//
//	kas_requests_total                → RequestEndEvent
//	kas_request_duration_seconds      → RequestEndEvent.Duration
//	kas_stage_duration_seconds        → StageEndEvent
//	kas_stage_deadline_exceeded_total → StageDeadlineEvent
//	kas_deadline_exceeded_total       → StageDeadlineEvent（deadline_stage）
//	kas_evidence_cache_total          → CacheEvent
//	kas_evidence_cache_evictions_total → CacheEvictionEvent
//	kas_evidence_filtered_total       → FilterEvent
//	kas_provider_errors_total         → ProviderErrorEvent
//	kas_decision_fallback_total       → FallbackEvent
//	kas_health_status                 → HealthEvent
//	trace 字段（config_version/provider/model/model_version/fallback_route/
//	cache_result/候选数量/过滤数量/reason code）→ 各事件公共元数据与字段
//
// 禁止字段（任何事件不得携带）：question 正文、Evidence 正文、身份明文
// （TenantKey/用户 ID/token）、secret、Evidence ID、trace ID 之外的自由字符串。
// TraceID 与 ConfigVersion 是允许的关联键。
type Event interface {
	Kind() string
}

// Meta 是事件公共元数据。
type Meta struct {
	TraceID       string
	ConfigVersion string
	At            time.Time
}

// RequestEndEvent 一次 Answer 请求结束（kas_requests_total / request_duration）。
type RequestEndEvent struct {
	Meta
	Outcome  string // ok / invalid_request / deadline_exceeded / evidence_unavailable / decision_unavailable / error / closed
	Action   string // 最终 action（枚举值）
	Duration time.Duration
	Degraded bool
}

// Kind implements Event.
func (RequestEndEvent) Kind() string { return "request_end" }

// StageEndEvent 一个阶段结束（kas_stage_duration_seconds）。
type StageEndEvent struct {
	Meta
	Stage     string // preliminary/first_round/rejudge/supplement/reserve
	Provider  string
	Outcome   string // ok / skipped / deadline_exceeded / provider_error / degraded
	Duration  time.Duration
	Remaining time.Duration // 阶段结束时剩余 deadline（trace 需求）
}

// Kind implements Event.
func (StageEndEvent) Kind() string { return "stage_end" }

// StageDeadlineEvent 阶段或 root deadline 超时
// （kas_stage_deadline_exceeded_total / kas_deadline_exceeded_total）。
type StageDeadlineEvent struct {
	Meta
	Stage         string // 阶段名或 "root"
	DeadlineStage string // deadline_stage label 值
}

// Kind implements Event.
func (StageDeadlineEvent) Kind() string { return "stage_deadline" }

// CacheEvent Evidence 缓存读写结果（kas_evidence_cache_total）。
type CacheEvent struct {
	Meta
	Result string // hit / miss / bypass / error
}

// Kind implements Event.
func (CacheEvent) Kind() string { return "cache" }

// CacheEvictionEvent 缓存淘汰（kas_evidence_cache_evictions_total）。
type CacheEvictionEvent struct {
	Meta
	Reason string // capacity_entries / capacity_bytes / item_too_large / namespace_delete
}

// Kind implements Event.
func (CacheEvictionEvent) Kind() string { return "cache_eviction" }

// FilterEvent 有效期/权限过滤（kas_evidence_filtered_total）。
type FilterEvent struct {
	Meta
	Reason string // expired / future / unauthorized / unknown_validity
	Count  int
}

// Kind implements Event.
func (FilterEvent) Kind() string { return "filter" }

// ProviderErrorEvent provider 错误（kas_provider_errors_total）。
type ProviderErrorEvent struct {
	Meta
	Provider   string // evidence / jev / rules / decision_cache
	Operation  string // search / decide / health
	ErrorClass string // timeout / canceled / unavailable / contract / closed
}

// Kind implements Event.
func (ProviderErrorEvent) Kind() string { return "provider_error" }

// FallbackEvent 决策降级链路由（kas_decision_fallback_total）。
type FallbackEvent struct {
	Meta
	Route  string // jev / deterministic_rules / decision_cache / abstain
	Reason string // 受控 reason code（domain.ReasonXxx）
}

// Kind implements Event.
func (FallbackEvent) Kind() string { return "fallback" }

// HealthEvent 组件健康变化（kas_health_status）。
type HealthEvent struct {
	Meta
	Component string // evidence_provider / jev / …（有界集合）
	Status    string // healthy / degraded / not_ready
}

// Kind implements Event.
func (HealthEvent) Kind() string { return "health" }

// DecisionTraceEvent 决策过程的 trace 字段（docs/design/04 §6：provider、
// model、model_version、候选数量、action、reason code）。
type DecisionTraceEvent struct {
	Meta
	Provider       string
	Model          string
	ModelVersion   string
	FallbackRoute  string
	CandidateCount int
	Action         string
	ReasonCode     string
	Confidence     float64
}

// Kind implements Event.
func (DecisionTraceEvent) Kind() string { return "decision_trace" }
