package kas

import (
	"context"
	"time"
)

// TODO(S5): Epic 5.1 落地；此处为 docs/design/03 §3 占位。

// CandidateGroup 是证据候选组。Attributes 仅允许配置白名单字段。
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

// Decision 是决策 provider 的输出。每个 TargetGroupID 必须来自输入；
// Confidence 出现 NaN/Inf 按 provider contract error 拒绝。
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
