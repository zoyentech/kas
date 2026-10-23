package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"sort"
	"strings"
)

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ScoreInput 是单例评分输入：期望用例 + 实际返回（引用按序）+ 耗时。
type ScoreInput struct {
	Case         Case
	Action       string
	Citations    []string // EvidenceID 有序列表
	Answer       string
	DurationMS   float64
	CacheHit     bool
	FallbackUsed bool
}

// CaseScore 是单例得分。
type CaseScore struct {
	CaseID           string
	Risk             string
	ActionCorrect    bool
	Recall           float64 // |expected ∩ cited| / |expected|
	Precision        float64 // |expected ∩ cited| / |cited|
	Rank             int     // 首个命中的期望证据排名（0=未命中）
	FactsCovered     float64 // 期望事实被回答覆盖比例
	Unanswerable     bool    // 不可答用例（无期望证据或 must_abstain）
	FactCheckable    bool    // 含事实断言（事实支持率只对这类用例平均）
	ForbiddenCited   int     // 引用了 forbidden 证据（P0 违规）
	ExpiredCited     int     // 过期证据被引用（P0 违规；场景套件保证为 0）
	AnswerNoEvidence bool    // 无据回答（P0 违规）
}

// SetScore 是全集/P0 子集聚合计分（docs/05 §4 指标）。
type SetScore struct {
	Count                  int
	RecallAt5              float64
	PrecisionAt5           float64
	MRR                    float64
	NDCG                   float64
	FactSupport            float64
	AbstainAccuracy        float64
	MedianMS, P95MS, P99MS float64
	P0Violations           int
}

// ScoreCase 评分单例。
func ScoreCase(in ScoreInput) CaseScore {
	s := CaseScore{CaseID: in.Case.CaseID, Risk: in.Case.Risk}
	s.ActionCorrect = in.Action == in.Case.ExpectedAction
	s.Unanswerable = in.Case.MustAbstain || len(in.Case.ExpectedEvidenceIDs) == 0
	s.FactCheckable = len(in.Case.ExpectedAnswerFacts) > 0 && !s.Unanswerable

	expected := setOf(in.Case.ExpectedEvidenceIDs)
	cited := setOf(in.Citations)

	hit := 0
	for id := range cited {
		if expected[id] {
			hit++
		}
	}
	if len(expected) > 0 {
		s.Recall = float64(hit) / float64(len(expected))
	}
	if len(cited) > 0 {
		s.Precision = float64(hit) / float64(len(cited))
	}
	for i, id := range in.Citations {
		if expected[id] {
			s.Rank = i + 1
			break
		}
	}
	forbidden := setOf(in.Case.ForbiddenEvidenceIDs)
	for _, id := range in.Citations {
		if forbidden[id] {
			s.ForbiddenCited++
		}
	}
	if len(in.Case.ExpectedAnswerFacts) > 0 {
		covered := 0
		for _, fact := range in.Case.ExpectedAnswerFacts {
			if strings.Contains(in.Answer, fact) {
				covered++
			}
		}
		s.FactsCovered = float64(covered) / float64(len(in.Case.ExpectedAnswerFacts))
	}
	if in.Action == "answer" && len(cited) == 0 {
		s.AnswerNoEvidence = true
	}
	return s
}

// Aggregate 聚合一组单例得分；durations 为对应毫秒时延（可为空）。
func Aggregate(scores []CaseScore, durations []float64) SetScore {
	out := SetScore{Count: len(scores)}
	if len(scores) == 0 {
		return out
	}
	var recallSum, precSum, mrrSum, ndcgSum, factSum float64
	var answerable, factCheckable float64
	abstainTotal, abstainCorrect := 0, 0
	for _, s := range scores {
		if s.Unanswerable {
			// 不可答用例不进入检索质量均值（docs/05 §4：分开统计）
			abstainTotal++
			if s.ActionCorrect {
				abstainCorrect++
			}
			if s.Risk == "p0" {
				out.P0Violations += s.ForbiddenCited + s.ExpiredCited
				if s.AnswerNoEvidence {
					out.P0Violations++
				}
			}
			continue
		}
		answerable++
		recallSum += s.Recall
		precSum += s.Precision
		if s.Rank > 0 {
			mrrSum += 1 / float64(s.Rank)
			ndcgSum += 1 / math.Log2(float64(s.Rank)+1)
		}
		if s.FactCheckable {
			factSum += s.FactsCovered
			factCheckable++
		}
		if s.Risk == "p0" {
			out.P0Violations += s.ForbiddenCited + s.ExpiredCited
			if s.AnswerNoEvidence {
				out.P0Violations++
			}
		}
	}
	if answerable > 0 {
		out.RecallAt5 = recallSum / answerable
		out.PrecisionAt5 = precSum / answerable
		out.MRR = mrrSum / answerable
		out.NDCG = ndcgSum / answerable
	}
	if factCheckable > 0 {
		out.FactSupport = factSum / factCheckable
	}
	if abstainTotal > 0 {
		out.AbstainAccuracy = float64(abstainCorrect) / float64(abstainTotal)
	}

	if len(durations) > 0 {
		ds := append([]float64(nil), durations...)
		sort.Float64s(ds)
		out.MedianMS = quantile(ds, 0.50)
		out.P95MS = quantile(ds, 0.95)
		out.P99MS = quantile(ds, 0.99)
	}
	return out
}

// quantile 就近秩分位（线性插值）。
func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi || hi >= len(sorted) {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func setOf(in []string) map[string]bool {
	m := make(map[string]bool, len(in))
	for _, v := range in {
		m[v] = true
	}
	return m
}
