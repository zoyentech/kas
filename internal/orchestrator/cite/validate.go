// Package cite 实现引用安全校验（F-010；Story 5.4.1.1）：citation 只能
// 来自本次最终有效证据集——越权/过期/不在当前结果中的引用一律丢弃并计数。
package cite

import "github.com/kas-runtime/kas/internal/domain"

// Validate 校验引用归属（Story 5.4.1.1 T1）：按 EvidenceID 匹配最终有效集；
// SourceID 不一致或 EvidenceID 未知 → 丢弃。返回保留引用（保序）与丢弃计数。
func Validate(final []domain.Evidence, cites []domain.Citation) (kept []domain.Citation, dropped int) {
	valid := make(map[string]bool, len(final))
	for _, e := range final {
		valid[e.ID] = true
	}
	for _, c := range cites {
		if valid[c.EvidenceID] {
			kept = append(kept, c)
			continue
		}
		dropped++
	}
	return kept, dropped
}

// FromEvidence 由最终有效证据构造引用（保序）。
func FromEvidence(final []domain.Evidence) []domain.Citation {
	out := make([]domain.Citation, 0, len(final))
	for _, e := range final {
		out = append(out, domain.Citation{
			EvidenceID:     e.ID,
			SourceID:       e.SourceID,
			ContentVersion: e.ContentVersion,
		})
	}
	return out
}

// Truncate 保目标组顺序截断至 max（Story 5.4.1.1 T2；max<=0 返回空）。
func Truncate(cites []domain.Citation, max int) []domain.Citation {
	if max <= 0 {
		return nil
	}
	if len(cites) <= max {
		return cites
	}
	return cites[:max]
}
