package rules_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/decision/rules"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func ptr(f float64) *float64 { return &f }

func sampleRules() []config.RuleConfig {
	return []config.RuleConfig{
		{Name: "no_valid_evidence", When: config.CondValidEvidenceZero, Action: "abstain"},
		{Name: "low_confidence", When: config.CondConfidenceBelowThreshold, Action: "ask_clarification", Threshold: ptr(0.6)},
	}
}

// TestRulesEvaluate docs/02 两条规则样例可加载求值（Story 5.1.2.1 T1）。
func TestRulesEvaluate(t *testing.T) {
	e, err := rules.New(sampleRules())
	if err != nil {
		t.Fatal(err)
	}
	withEvidence := domain.DecisionInput{Groups: []domain.CandidateGroup{
		{ID: "g1", Evidence: []domain.EvidenceRef{{EvidenceID: "e1"}}},
	}}
	noEvidence := domain.DecisionInput{Groups: []domain.CandidateGroup{{ID: "g1"}}}

	// 无证据 → no_valid_evidence 命中 abstain
	action, name, matched := e.Evaluate(rules.Context{Input: noEvidence})
	if !matched || action != domain.ActionAbstain || name != "no_valid_evidence" {
		t.Fatalf("无证据应命中 no_valid_evidence: %v %s %s", matched, action, name)
	}
	// 有证据、无上游 → 不命中
	_, _, matched = e.Evaluate(rules.Context{Input: withEvidence})
	if matched {
		t.Fatal("有证据且无上游置信度不应命中")
	}
	// 有证据 + 上游低置信 → low_confidence 命中 clarify
	prior := domain.Decision{Action: domain.ActionAnswer, Confidence: 0.3}
	action, name, matched = e.Evaluate(rules.Context{Input: withEvidence, Prior: &prior})
	if !matched || action != domain.ActionClarify || name != "low_confidence" {
		t.Fatalf("低置信应命中 low_confidence: %v %s %s", matched, action, name)
	}
	// 高置信不命中
	prior2 := domain.Decision{Action: domain.ActionAnswer, Confidence: 0.9}
	_, _, matched = e.Evaluate(rules.Context{Input: withEvidence, Prior: &prior2})
	if matched {
		t.Fatal("高置信不应命中")
	}
}

// TestRulesPriorityStable 声明序首个命中生效；重复调用输出一致（确定性）
// （Story 5.1.2.1 T2；docs/05 §2 冲突行）。
func TestRulesPriorityStable(t *testing.T) {
	cfg := []config.RuleConfig{
		{Name: "first", When: config.CondValidEvidenceZero, Action: "abstain"},
		{Name: "second", When: config.CondValidEvidenceZero, Action: "ask_clarification"}, // 同条件后声明
	}
	e, _ := rules.New(cfg)
	in := domain.DecisionInput{}
	for i := 0; i < 5; i++ {
		action, name, matched := e.Evaluate(rules.Context{Input: in})
		if !matched || name != "first" || action != domain.ActionAbstain {
			t.Fatalf("冲突裁决不稳定: %s %s", name, action)
		}
	}
}

// TestRulesUnknownCondition 构建期拒绝未知条件。
func TestRulesUnknownCondition(t *testing.T) {
	_, err := rules.New([]config.RuleConfig{{Name: "r", When: "no_such_cond", Action: "abstain"}})
	if err == nil {
		t.Fatal("未知条件应构建期报错")
	}
	_, err = rules.New([]config.RuleConfig{{Name: "r", When: config.CondConfidenceBelowThreshold, Action: "abstain"}})
	if err == nil {
		t.Fatal("缺阈值应构建期报错")
	}
}

// TestRulesProvider Decide 适配 + ErrNoMatch。
func TestRulesProvider(t *testing.T) {
	p, err := rules.NewProvider(sampleRules())
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.Decide(context.Background(), domain.DecisionInput{})
	if err != nil || d.Action != domain.ActionAbstain || d.Provider != "rules" || !d.Degraded {
		t.Fatalf("Decide 适配异常: %+v %v", d, err)
	}
	_, err = p.Decide(context.Background(), domain.DecisionInput{
		Groups: []domain.CandidateGroup{{Evidence: []domain.EvidenceRef{{}}}},
	})
	if !errors.Is(err, rules.ErrNoMatch) {
		t.Fatalf("未命中应返回 ErrNoMatch: %v", err)
	}
}

// TestPreliminaryJudge 初判挂载：无证据规则 → abstain；否则 continue。
func TestPreliminaryJudge(t *testing.T) {
	j, err := rules.NewPreliminary(sampleRules())
	if err != nil {
		t.Fatal(err)
	}
	action, _, matched := j.Judge(context.Background(), domain.DecisionInput{})
	if !matched || action != domain.ActionAbstain {
		t.Fatalf("初判应命中 abstain: %v %s", matched, action)
	}
	_, _, matched = j.Judge(context.Background(), domain.DecisionInput{
		Groups: []domain.CandidateGroup{{Evidence: []domain.EvidenceRef{{}}}},
	})
	if matched {
		t.Fatal("有证据初判应 continue")
	}
}
