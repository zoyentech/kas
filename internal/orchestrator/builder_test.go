package orchestrator_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/orchestrator"
	"github.com/kas-runtime/kas/internal/orchestrator/cite"
)

func validEvidence(ids ...string) domain.EvidenceResult {
	var ev []domain.Evidence
	for _, id := range ids {
		ev = append(ev, domain.Evidence{ID: id, SourceID: "src-" + id, ContentVersion: "v1"})
	}
	return domain.EvidenceResult{Evidence: ev}
}

func answerCfg() config.AnswerConfig {
	return config.AnswerConfig{MaxCitations: 5, RequireCitationForFactualClaims: true,
		AllowAnswerWithoutEvidence: false, OutputOnUnknown: "abstain"}
}

// TestCiteValidate 引用归属校验（Story 5.4.1.1 T1）：混合引用集仅有效保留。
func TestCiteValidate(t *testing.T) {
	final := []domain.Evidence{
		{ID: "ok1", SourceID: "s1"}, {ID: "ok2", SourceID: "s2"},
	}
	cites := []domain.Citation{
		{EvidenceID: "ok1", SourceID: "s1"},
		{EvidenceID: "stale", SourceID: "sX"}, // 不在最终集
		{EvidenceID: "ok2", SourceID: "s2"},
		{EvidenceID: "ghost", SourceID: "s?"}, // 未知 ID
	}
	kept, dropped := cite.Validate(final, cites)
	if len(kept) != 2 || kept[0].EvidenceID != "ok1" || kept[1].EvidenceID != "ok2" {
		t.Fatalf("应保序保留有效引用: %+v", kept)
	}
	if dropped != 2 {
		t.Fatalf("丢弃计数 = %d, want 2", dropped)
	}
}

// TestCiteTruncate 保序截断（1/恰上限/超上限）。
func TestCiteTruncate(t *testing.T) {
	cites := []domain.Citation{{EvidenceID: "a"}, {EvidenceID: "b"}, {EvidenceID: "c"}}
	if got := cite.Truncate(cites, 1); len(got) != 1 || got[0].EvidenceID != "a" {
		t.Fatalf("截断 1 应保序取首: %+v", got)
	}
	if got := cite.Truncate(cites, 3); len(got) != 3 {
		t.Fatalf("恰上限不截断: %+v", got)
	}
	if got := cite.Truncate(cites, 2); len(got) != 2 || got[1].EvidenceID != "b" {
		t.Fatalf("超上限保序截断: %+v", got)
	}
	if got := cite.Truncate(cites, 0); got != nil {
		t.Fatalf("max=0 返回空: %+v", got)
	}
}

// TestBuilderAnswerWithCitations answer 组装：目标组引用 + 元数据完备
// （Story 5.4.1.2 T1）。
func TestBuilderAnswerWithCitations(t *testing.T) {
	b := orchestrator.NewBuilder(answerCfg())
	resp, err := b.Assemble(context.Background(), orchestrator.AssembleInput{
		Evidence: validEvidence("e1", "e2"),
		Groups: []domain.CandidateGroup{{ID: "g1", Evidence: []domain.EvidenceRef{
			{EvidenceID: "e1", SourceID: "src-e1", ContentVersion: "v1"},
			{EvidenceID: "e2", SourceID: "src-e2", ContentVersion: "v1"},
		}}},
		Decision:      domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9, AnswerText: "回答正文"},
		TraceID:       "t1",
		ConfigVersion: "cv1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != domain.ActionAnswer || resp.Answer != "回答正文" {
		t.Fatalf("answer 组装异常: %+v", resp)
	}
	if len(resp.Citations) != 2 || resp.Citations[0].EvidenceID != "e1" {
		t.Fatalf("引用应来自目标组且保序: %+v", resp.Citations)
	}
	if resp.Confidence == nil || *resp.Confidence != 0.9 {
		t.Fatal("Confidence 透传")
	}
	if resp.TraceID != "t1" || resp.ConfigVersion != "cv1" {
		t.Fatal("TraceID/ConfigVersion 注入")
	}
}

// TestBuilderNoEvidenceTerminal 无证据终态：output_on_unknown=abstain/clarify
// 与决策为 clarify 的透传（Story 5.4.1.1 T3）。
func TestBuilderNoEvidence(t *testing.T) {
	// 无证据 + answer 决策 → abstain（默认 output_on_unknown）
	b := orchestrator.NewBuilder(answerCfg())
	resp, err := b.Assemble(context.Background(), orchestrator.AssembleInput{
		Decision: domain.Decision{Action: domain.ActionAnswer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != domain.ActionAbstain {
		t.Fatalf("无证据 answer 应转 abstain: %+v", resp)
	}
	// output_on_unknown=ask_clarification
	cfg2 := answerCfg()
	cfg2.OutputOnUnknown = "ask_clarification"
	resp2, _ := orchestrator.NewBuilder(cfg2).Assemble(context.Background(), orchestrator.AssembleInput{
		Decision: domain.Decision{Action: domain.ActionAnswer},
	})
	if resp2.Action != domain.ActionClarify {
		t.Fatalf("无证据 + clarify 配置应转 clarify: %+v", resp2)
	}
	// 决策本身就是 clarify：尊重透传
	resp3, _ := orchestrator.NewBuilder(answerCfg()).Assemble(context.Background(), orchestrator.AssembleInput{
		Decision: domain.Decision{Action: domain.ActionClarify},
	})
	if resp3.Action != domain.ActionClarify {
		t.Fatalf("clarify 决策应透传: %+v", resp3)
	}
}

// TestBuilderZeroCitationAnswerIntercepted 零引用 answer 被终检拦截
// （Story 5.4.1.2 T2；P0"无据回答=0"）。
func TestBuilderZeroCitationAnswerIntercepted(t *testing.T) {
	b := orchestrator.NewBuilder(answerCfg())
	// 有证据但目标组引用全部不在最终集（模拟越权引用被丢弃）
	resp, err := b.Assemble(context.Background(), orchestrator.AssembleInput{
		Evidence: validEvidence("e1"),
		Groups: []domain.CandidateGroup{{ID: "g1", Evidence: []domain.EvidenceRef{
			{EvidenceID: "ghost"}, // 不在最终集
		}}},
		Decision: domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != domain.ActionAbstain || !resp.Degraded {
		t.Fatalf("零引用 answer 应被终检拦截: %+v", resp)
	}
	if !containsStr(resp.Degradation, "citation_missing") {
		t.Fatalf("reason 应含 citation_missing: %v", resp.Degradation)
	}
}

// TestBuilderMaxCitationsTruncated 超上限截断。
func TestBuilderMaxCitationsTruncated(t *testing.T) {
	cfg := answerCfg()
	cfg.MaxCitations = 2
	b := orchestrator.NewBuilder(cfg)
	resp, err := b.Assemble(context.Background(), orchestrator.AssembleInput{
		Evidence: validEvidence("e1", "e2", "e3"),
		Decision: domain.Decision{Action: domain.ActionAnswer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Citations) != 2 || resp.Citations[0].EvidenceID != "e1" || resp.Citations[1].EvidenceID != "e2" {
		t.Fatalf("应保序截断到 2: %+v", resp.Citations)
	}
}

// TestBuilderPureConversationMode 纯对话模式开关允许无引用 answer。
func TestBuilderPureConversationMode(t *testing.T) {
	cfg := answerCfg()
	cfg.AllowAnswerWithoutEvidence = true
	b := orchestrator.NewBuilder(cfg)
	resp, err := b.Assemble(context.Background(), orchestrator.AssembleInput{
		Evidence: validEvidence("e1"),
		Groups: []domain.CandidateGroup{{ID: "g1", Evidence: []domain.EvidenceRef{
			{EvidenceID: "ghost"},
		}}},
		Decision: domain.Decision{Action: domain.ActionAnswer},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != domain.ActionAnswer {
		t.Fatalf("纯对话模式应允许 answer: %+v", resp)
	}
}

// TestBuilderReasonsDedup reason 去重保序。
func TestBuilderReasonsDedup(t *testing.T) {
	b := orchestrator.NewBuilder(answerCfg())
	resp, err := b.Assemble(context.Background(), orchestrator.AssembleInput{
		Evidence: validEvidence("e1"), // 隔离 no_valid_evidence 分支，专注去重
		Reasons:  []string{"jev_unavailable", "jev_unavailable", "rules_applied"},
		Decision: domain.Decision{Action: domain.ActionAbstain},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resp.Degradation, []string{"jev_unavailable", "rules_applied"}) {
		t.Fatalf("reason 去重异常: %v", resp.Degradation)
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
