package orchestrator

import (
	"context"
)

// DefaultPreliminary 是初判默认实现：放行检索路径（continue）。
// 规则引擎初判挂载见 internal/decision/rules（PreliminaryJudge 适配），
// 由装配方按需注入替换。
type DefaultPreliminary struct{}

// Judge 恒返回 continue（占位）。
func (DefaultPreliminary) Judge(ctx context.Context, in PreliminaryInput) (PreliminaryOutcome, error) {
	return PreliminaryOutcome{Kind: PreliminaryContinue}, nil
}
