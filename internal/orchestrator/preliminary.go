package orchestrator

import (
	"context"

	"github.com/kas-runtime/kas/internal/domain"
)

// DefaultPreliminary 是初判默认实现：放行检索路径（continue）。
// S5 规则引擎落地后替换（Story 3.3.1.1 T2 的替换点）。
type DefaultPreliminary struct{}

// Judge 恒返回 continue（占位）。
func (DefaultPreliminary) Judge(ctx context.Context, in PreliminaryInput) (PreliminaryOutcome, error) {
	return PreliminaryOutcome{Kind: PreliminaryContinue}, nil
}

// DefaultAssembler 是最小组装器（Story 3.3.1.1 骨架）：
//   - 有决策：透传 action；answer/supplement 由 S5 补充引用校验
//   - 无决策（决策失败/degraded）：abstain + 降级 reason
//
// S5 的 citation 安全校验（Story 5.4.1）替换本实现。
type DefaultAssembler struct{}

// Assemble 组装响应骨架：config_version/trace 恒定，降级标记与受控 reason 透传。
func (DefaultAssembler) Assemble(ctx context.Context, in AssembleInput) (domain.Response, error) {
	resp := domain.Response{
		TraceID:       in.TraceID,
		ConfigVersion: in.ConfigVersion,
		Degraded:      in.Degraded,
		Degradation:   in.Reasons,
	}
	if in.Decision.Action != "" {
		resp.Action = in.Decision.Action
		if in.Decision.Confidence != 0 {
			c := in.Decision.Confidence
			resp.Confidence = &c
		}
	} else {
		resp.Action = domain.ActionAbstain
		if len(in.Reasons) == 0 {
			resp.Degradation = append(resp.Degradation, domain.ReasonNoValidEvidence)
		}
	}
	return resp, nil
}
