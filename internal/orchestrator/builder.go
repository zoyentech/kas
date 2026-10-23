package orchestrator

import (
	"context"

	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/orchestrator/cite"
)

// Builder 是回答组装器（Story 5.4.1.2；reserve 阶段执行，计入 reserve
// 预算）：目标组 + 已验证引用 + Decision 元数据 → Response。纯转换无 IO。
//
// 安全闸（最后一道，docs/design/03 §2）：
//  1. citation ⊆ 最终有效 Evidence（cite.Validate）
//  2. answer 必须含 ≥1 条已验证引用（allow_answer_without_evidence=false
//     时；零引用 answer 被终检拦截 → abstain + citation_missing）
//  3. 无有效证据 → 按 output_on_unknown 转 clarify/abstain（docs/01 §6
//     "Evidence 不完整或过期时只能澄清/拒答"）
type Builder struct {
	Cfg config.AnswerConfig
}

// NewBuilder 构建组装器。
func NewBuilder(cfg config.AnswerConfig) *Builder { return &Builder{Cfg: cfg} }

// Assemble 组装 Response。
func (b *Builder) Assemble(ctx context.Context, in AssembleInput) (domain.Response, error) {
	resp := domain.Response{
		TraceID:       in.TraceID,
		ConfigVersion: in.ConfigVersion,
		Degraded:      in.Degraded,
		Degradation:   dedupReasons(in.Reasons),
	}

	final := in.Evidence.Evidence
	resp.Action = in.Decision.Action

	// 无有效证据：answer/supplement 类动作不可执行（无据禁答），按
	// output_on_unknown 转 clarify/abstain（docs/01 §6"只能澄清/拒答"）；
	// 决策本身就是 clarify/abstain 时尊重透传
	if len(final) == 0 {
		switch resp.Action {
		case domain.ActionClarify, domain.ActionAbstain:
			resp.Degradation = appendReason(resp.Degradation, domain.ReasonNoValidEvidence)
			return resp, nil
		default:
			if b.Cfg.OutputOnUnknown == "ask_clarification" {
				resp.Action = domain.ActionClarify
			} else {
				resp.Action = domain.ActionAbstain
			}
			resp.Degradation = appendReason(resp.Degradation, domain.ReasonNoValidEvidence)
			return resp, nil
		}
	}

	switch in.Decision.Action {
	case domain.ActionAnswer, domain.ActionSupplement:
		// 引用来自目标组的证据（保目标组顺序），校验归属 + 截断
		refs := targetRefs(in.Decision, in.Groups, final)
		// 越权/过期/未知 ID 的引用被丢弃（docs/04 §5 无 citation 指标，
		// 丢弃体现为 abstain + citation_missing，不另发事件）
		valid, _ := cite.Validate(final, refs)
		valid = cite.Truncate(valid, b.Cfg.MaxCitations)
		// 终检：answer 无引用 → 降级 abstain（最后一道闸）
		if len(valid) == 0 && !b.Cfg.AllowAnswerWithoutEvidence {
			resp.Action = domain.ActionAbstain
			resp.Degraded = true
			resp.Degradation = appendReason(resp.Degradation, ReasonCitationMissing)
			return resp, nil
		}
		resp.Citations = valid
		resp.Answer = in.Decision.AnswerText
		if in.Decision.Confidence > 0 {
			c := in.Decision.Confidence
			resp.Confidence = &c
		}
	default:
		// clarify/abstain：无正文输出
		resp.Answer = in.Decision.AnswerText
	}
	return resp, nil
}

// targetRefs 按目标组顺序取证据引用（目标组 ID 已由 ValidateDecision
// 保证属于输入候选）。
func targetRefs(dec domain.Decision, groups []domain.CandidateGroup, final []domain.Evidence) []domain.Citation {
	if len(dec.TargetGroupIDs) == 0 {
		// 未指定目标组：全部最终证据（保序）
		return cite.FromEvidence(final)
	}
	byID := make(map[string]bool, len(dec.TargetGroupIDs))
	for _, id := range dec.TargetGroupIDs {
		byID[id] = true
	}
	var out []domain.Citation
	for _, g := range groups {
		if !byID[g.ID] {
			continue
		}
		for _, ref := range g.Evidence {
			out = append(out, domain.Citation{
				EvidenceID:     ref.EvidenceID,
				SourceID:       ref.SourceID,
				ContentVersion: ref.ContentVersion,
			})
		}
	}
	return out
}

// ReasonCitationMissing 是零引用 answer 被终检拦截的 reason code。
const ReasonCitationMissing = "citation_missing"

func dedupReasons(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, r := range in {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}

func appendReason(reasons []string, reason string) []string {
	for _, r := range reasons {
		if r == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}
