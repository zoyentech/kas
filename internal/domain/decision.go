package domain

import (
	"context"
	"time"
)

// CandidateGroup 是证据候选组。Attributes 仅允许配置白名单字段（防高基数
// 与敏感信息渗漏）。
type CandidateGroup struct {
	ID         string
	Evidence   []EvidenceRef
	Score      float64
	Attributes map[string]string
}

// DecisionInput 是决策 provider 的输入。
type DecisionInput struct {
	Question      string
	Groups        []CandidateGroup
	ValidAt       time.Time
	ConfigVersion string
	ModelHint     string
}

// Decision 是决策 provider 的输出。每个 TargetGroupID 必须来自输入
// Groups；Confidence 出现 NaN/Inf 按 provider contract error 拒绝。
type Decision struct {
	Action         ActionType
	TargetGroupIDs []string
	NeedSupplement bool
	ReasonCode     string
	Confidence     float64
	Provider       string
	Model          string
	ModelVersion   string
	Degraded       bool
}

// DecisionProvider 将候选组映射为有限 action；Jev 私有协议只存在于
// Jev adapter，编排器不直接依赖（F-007）。
type DecisionProvider interface {
	Decide(ctx context.Context, in DecisionInput) (Decision, error)
	Health(ctx context.Context) ComponentHealth
}

// ComponentHealth 是组件自报健康状态（EvidenceProvider/DecisionProvider/
// 宿主注入检查点共用）。
type ComponentHealth struct {
	Status ComponentStatus
	// Detail 仅进入安全日志/trace，不进入指标 label（基数防护）。
	Detail string
}

// ComponentStatus 是组件健康枚举。
type ComponentStatus string

const (
	ComponentHealthy  ComponentStatus = "healthy"
	ComponentDegraded ComponentStatus = "degraded"
	ComponentNotReady ComponentStatus = "not_ready"
	ComponentUnknown  ComponentStatus = "unknown"
)

// HealthReport 是 Client.Health 的聚合结果（F-005；S6 实现聚合逻辑）。
type HealthReport struct {
	Status     OverallStatus
	Components map[string]ComponentHealth
	CheckedAt  time.Time
}

// OverallStatus 是总健康状态。
type OverallStatus string

const (
	StatusHealthy  OverallStatus = "healthy"
	StatusDegraded OverallStatus = "degraded"
	StatusNotReady OverallStatus = "not_ready"
)

// HealthChecks 是宿主注入的额外健康检查点（组件名 → 检查函数）。
type HealthChecks map[string]func(ctx context.Context) ComponentHealth
