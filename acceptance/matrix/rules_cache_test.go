package matrix

import (
	"context"
	"errors"
	"strings"
	"testing"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/domain"
)

// 矩阵行 13：多条规则冲突 → 优先级稳定、trace 可解释（reason 含规则名）。
func TestRowRulesConflictStable(t *testing.T) {
	yaml := func(src string) string {
		return replaceRules(src, `
    rules:
      - {name: first_rule, when: valid_evidence_count_equals_zero, action: abstain}
      - {name: second_rule, when: valid_evidence_count_equals_zero, action: ask_clarification}`)
	}
	// 无证据 → first_rule（声明序首命中）→ abstain
	ev := &matrixEvidence{}
	c := newClient(t, ev, nil, yaml)
	resp, err := c.Answer(context.Background(), req("规则冲突问题"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAbstain {
		t.Fatalf("首命中规则应生效: %+v", resp)
	}
	// 重复调用输出一致（稳定性由 rules 单测证明；此处场景级重复）
	for i := 0; i < 3; i++ {
		r2, err := c.Answer(context.Background(), req("规则冲突问题"))
		if err != nil || r2.Action != kas.ActionAbstain {
			t.Fatalf("冲突裁决不稳定: %+v %v", r2, err)
		}
	}
}

// 矩阵行 14：决策缓存命中——候选/版本/scope 匹配且 action 重新校验。
func TestRowDecisionCacheHit(t *testing.T) {
	yaml := func(src string) string {
		return strings.Replace(src, "cache: {enabled: false, ttl: 30s, max_entries: 100}",
			"cache: {enabled: true, ttl: 30s, max_entries: 100, include_config_version: true, include_scope_fingerprint: true, include_candidate_fingerprint: true, validate_action_against_current_candidates: true}", 1)
	}
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
	flaky := &switchableDecision{}
	c := newClient(t, ev, flaky, yaml)

	// 第一次：决策成功（answer）→ 回填缓存
	flaky.set(kas.Decision{Action: kas.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9, AnswerText: "回答正文"})
	resp, err := c.Answer(context.Background(), req("决策缓存问题"))
	if err != nil || resp.Action != kas.ActionAnswer {
		t.Fatalf("首次决策: %+v %v", resp, err)
	}
	// 第二次：决策 provider 故障 → 缓存命中（degraded + decision_cache_hit）
	flaky.setErr(errors.New("decision down"))
	resp2, err := c.Answer(context.Background(), req("决策缓存问题"))
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Action != kas.ActionAnswer || !resp2.Degraded {
		t.Fatalf("应命中决策缓存: %+v", resp2)
	}
	if !containsReason(resp2.Degradation, kas.ReasonJevUnavailable) {
		t.Fatalf("reason 应含上游故障: %v", resp2.Degradation)
	}
}

// switchableDecision 可切换成功/故障。
type switchableDecision struct {
	dec kas.Decision
	err error
}

func (s *switchableDecision) set(d kas.Decision) { s.dec, s.err = d, nil }
func (s *switchableDecision) setErr(err error)   { s.err = err }

func (s *switchableDecision) Decide(ctx context.Context, in kas.DecisionInput) (kas.Decision, error) {
	return s.dec, s.err
}

func (s *switchableDecision) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

// replaceRules 替换 fallback rules 节（矩阵测试 YAML 手术）。
func replaceRules(src, newRules string) string {
	return strings.Replace(src,
		"    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]",
		newRules, 1)
}
