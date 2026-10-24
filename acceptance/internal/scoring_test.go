package internal_test

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	acc "github.com/zoyentech/kas/acceptance/internal"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestScoringFormulas 手工构造小金集验证公式（Story 7.1.1.2 T2）。
func TestScoringFormulas(t *testing.T) {
	// 用例：期望 [a,b]；实际引用 [x,a,b]（x 是干扰）
	c := acc.Case{
		CaseID: "golden-9001", Risk: "p0", Question: "q",
		PrincipalScope: "s", KnowledgeNamespace: "ns", CorpusGeneration: "g",
		ValidAt: at(t, "2026-10-23T00:00:00Z"), ExpectedAction: "answer",
		ExpectedEvidenceIDs: []string{"a", "b"},
		ExpectedAnswerFacts: []string{"事实一", "事实二"},
	}
	s := acc.ScoreCase(acc.ScoreInput{
		Case: c, Action: "answer", Citations: []string{"x", "a", "b"}, Answer: "回答含 事实一",
	})
	if s.Recall != 1.0 {
		t.Fatalf("recall = %v, want 1.0", s.Recall)
	}
	if s.Precision != 2.0/3.0 {
		t.Fatalf("precision = %v, want %.4f", s.Precision, 2.0/3.0)
	}
	if s.Rank != 2 {
		t.Fatalf("rank = %d, want 2", s.Rank)
	}
	if s.FactsCovered != 0.5 {
		t.Fatalf("facts = %v, want 0.5", s.FactsCovered)
	}
	if s.ActionCorrect != true || s.Unanswerable || s.ForbiddenCited != 0 {
		t.Fatalf("判定异常: %+v", s)
	}

	// 未命中：rank 0，recall 0
	s2 := acc.ScoreCase(acc.ScoreInput{
		Case: c, Action: "answer", Citations: []string{"x"}, Answer: "",
	})
	if s2.Rank != 0 || s2.Recall != 0 {
		t.Fatalf("未命中判定异常: %+v", s2)
	}
	// 无据回答：P0 违规
	s3 := acc.ScoreCase(acc.ScoreInput{
		Case: c, Action: "answer", Citations: nil, Answer: "编造",
	})
	if !s3.AnswerNoEvidence {
		t.Fatal("无据回答应计违规")
	}
	// forbidden 引用
	c2 := c
	c2.ForbiddenEvidenceIDs = []string{"a"}
	s4 := acc.ScoreCase(acc.ScoreInput{Case: c2, Action: "answer", Citations: []string{"a"}})
	if s4.ForbiddenCited != 1 {
		t.Fatal("forbidden 引用应计数")
	}
}

// TestAggregateFormulas 聚合公式（MRR/nDCG/分位）。
func TestAggregateFormulas(t *testing.T) {
	base := acc.Case{
		Risk: "p0", PrincipalScope: "s", KnowledgeNamespace: "ns", CorpusGeneration: "g",
		ValidAt: at(t, "2026-10-23T00:00:00Z"), ExpectedAction: "answer",
		ExpectedEvidenceIDs: []string{"a"}, ExpectedAnswerFacts: []string{"f"},
	}
	scores := []acc.CaseScore{
		acc.ScoreCase(acc.ScoreInput{Case: base, Action: "answer", Citations: []string{"a"}, Answer: "f"}),
		func() acc.CaseScore {
			c := base
			c.CaseID = "g2"
			return acc.ScoreCase(acc.ScoreInput{Case: c, Action: "answer", Citations: []string{"x", "a"}, Answer: "f"})
		}(),
	}
	// MRR = (1/1 + 1/2)/2 = 0.75
	agg := acc.Aggregate(scores, []float64{10, 20, 30})
	if math.Abs(agg.MRR-0.75) > 1e-9 {
		t.Fatalf("MRR = %v, want 0.75", agg.MRR)
	}
	// nDCG = (1/log2(2) + 1/log2(3))/2 = (1 + 0.6309)/2
	wantNDCG := (1.0 + 1/math.Log2(3)) / 2
	if math.Abs(agg.NDCG-wantNDCG) > 1e-9 {
		t.Fatalf("nDCG = %v, want %v", agg.NDCG, wantNDCG)
	}
	if agg.RecallAt5 != 1.0 || agg.FactSupport != 1.0 {
		t.Fatalf("recall/fact 异常: %+v", agg)
	}
	// 分位：median 20 / p95 接近 30
	if agg.MedianMS != 20 {
		t.Fatalf("median = %v", agg.MedianMS)
	}
	if agg.P95MS <= 20 || agg.P95MS > 30 {
		t.Fatalf("p95 = %v", agg.P95MS)
	}
}

// TestAggregateAbstainAndP0 不可答集单独统计 + P0 违规计数。
func TestAggregateAbstainAndP0(t *testing.T) {
	answerable := acc.Case{
		Risk: "p0", PrincipalScope: "s", KnowledgeNamespace: "ns", CorpusGeneration: "g",
		ValidAt: at(t, "2026-10-23T00:00:00Z"), ExpectedAction: "answer",
		ExpectedEvidenceIDs: []string{"a"}, ExpectedAnswerFacts: []string{"f"},
	}
	unanswerable := answerable
	unanswerable.ExpectedEvidenceIDs = nil
	unanswerable.MustAbstain = true
	unanswerable.ExpectedAnswerFacts = nil
	unanswerable.ExpectedAction = "abstain"

	scores := []acc.CaseScore{
		acc.ScoreCase(acc.ScoreInput{Case: answerable, Action: "answer", Citations: []string{"a"}, Answer: "f"}),
		acc.ScoreCase(acc.ScoreInput{Case: unanswerable, Action: "abstain"}),
		acc.ScoreCase(acc.ScoreInput{Case: unanswerable, Action: "answer", Citations: nil, Answer: "x"}), // 违规
	}
	agg := acc.Aggregate(scores, nil)
	if agg.AbstainAccuracy != 0.5 {
		t.Fatalf("abstain accuracy = %v, want 0.5", agg.AbstainAccuracy)
	}
	if agg.P0Violations != 1 { // 无据回答
		t.Fatalf("P0 violations = %d", agg.P0Violations)
	}
	// recall 只对可答用例平均：=1.0
	if agg.RecallAt5 != 1.0 {
		t.Fatalf("recall 应排除不可答: %v", agg.RecallAt5)
	}
}

// TestLoaderValidation 加载期校验（Story 7.1.1.1 T1）。
func TestLoaderValidation(t *testing.T) {
	bad := []acc.Case{
		{
			CaseID: "bad-id", Risk: "p0", Question: "q", PrincipalScope: "s",
			KnowledgeNamespace: "ns", CorpusGeneration: "g",
			ValidAt: time.Now(), ExpectedAction: "answer",
		}, // case_id 格式
		{
			CaseID: "golden-9999", Risk: "p9", Question: "q", PrincipalScope: "s",
			KnowledgeNamespace: "ns", CorpusGeneration: "g",
			ValidAt: time.Now(), ExpectedAction: "answer",
		}, // risk
		{
			CaseID: "golden-9999", Risk: "p0", Question: " ", PrincipalScope: "s",
			KnowledgeNamespace: "ns", CorpusGeneration: "g",
			ValidAt: time.Now(), ExpectedAction: "answer",
		}, // 空 question
		{
			CaseID: "golden-9999", Risk: "p0", Question: "q", PrincipalScope: "s",
			KnowledgeNamespace: "ns", CorpusGeneration: "g",
			ValidAt: time.Now(), ExpectedAction: "explode",
		}, // action
		{
			CaseID: "golden-9999", Risk: "p0", Question: "q", PrincipalScope: "s",
			KnowledgeNamespace: "ns", CorpusGeneration: "g",
			ValidAt: time.Now(), ExpectedAction: "answer",
			Tags: []string{"bogus"},
		}, // tag
		{
			CaseID: "golden-9999", Risk: "p0", Question: "q", PrincipalScope: "s",
			KnowledgeNamespace: "ns", CorpusGeneration: "g",
			ValidAt: time.Now(), ExpectedAction: "answer",
			MustAbstain: true, ExpectedEvidenceIDs: []string{"a"},
		}, // 互斥
	}
	for i, b := range bad {
		if err := validateForTest(t, b); err == nil {
			t.Errorf("用例 %d 应被拒绝", i)
		}
	}
}

// validateForTest 写临时目录复用加载校验。
func validateForTest(t *testing.T, c acc.Case) error {
	t.Helper()
	raw, err := json.Marshal([]acc.Case{c})
	if err != nil {
		return err
	}
	path := filepath.Join(t.TempDir(), "case.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return err
	}
	_, err = acc.LoadDir(filepath.Dir(path))
	return err
}
