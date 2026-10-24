// Package matrix 是 docs/design/05 §2 验收矩阵的 19 类场景套件
// （Stage 7；行 ↔ 测试映射见 matrix_map.md）。
package matrix

import (
	"context"
	"strings"
	"testing"
	"time"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

const baseYAML = `
schema_version: 1
timeouts: {total: 400ms, preliminary: 40ms, first_round: 120ms, rejudge: 80ms, supplement: 100ms, reserve: 60ms, min_reserve: 40ms}
evidence:
  provider: matrix
  namespace: ns-a
  top_k: 4
  max_candidate_groups: 2
  validity:
    require_validity_policy_for_each_source: false
    default_max_age: {manual: 24h}
    exclude_future_dated: true
    fail_closed_on_unknown_validity: true
  cache:
    enabled: true
    implementation: memory
    max_entries: 100
    max_bytes: 8388608
    max_item_bytes: 1048576
    ttl: 60s
    include: [normalized_question, config_version, principal_scope_fingerprint, knowledge_namespace, corpus_generation, authorization_epoch]
decision:
  provider: jev
  jev: {endpoint_env: KAS_MTX_JEV_ENDPOINT, api_key_env: KAS_MTX_JEV_API_KEY, model: m, model_version: v1, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms, degraded_when_optional_provider_unavailable: true}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
`

func req(q string) kas.Request {
	return kas.Request{
		Question:  q,
		Principal: kas.PrincipalScope{TenantKey: "t", ScopeFingerprint: "fp-a", AuthorizationEpoch: "e1"},
		Knowledge: kas.KnowledgeScope{Namespace: "ns-a", CorpusGeneration: "g1"},
	}
}

// matrixEvidence 可编程语料（按问题前缀返回不同形态证据）。
type matrixEvidence struct {
	items []domain.Evidence
	err   error
	block bool
}

func (p *matrixEvidence) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	if p.block {
		<-ctx.Done()
		return kas.EvidenceResult{}, ctx.Err()
	}
	if p.err != nil {
		return kas.EvidenceResult{}, p.err
	}
	return kas.EvidenceResult{Evidence: p.items}, nil
}

func (p *matrixEvidence) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

func validEvidence(id string) domain.Evidence {
	until := time.Now().Add(time.Hour)
	return domain.Evidence{ID: id, SourceID: "s-" + id, ValidUntil: &until, ContentVersion: "v1", Score: 0.9}
}

func newClient(t *testing.T, ev kas.EvidenceProvider, dec kas.DecisionProvider, mutate func(string) string) *kas.Client {
	t.Helper()
	t.Setenv("KAS_MTX_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_MTX_JEV_API_KEY", "k")
	yamlSrc := baseYAML
	if mutate != nil {
		yamlSrc = mutate(yamlSrc)
	}
	cfg, err := kas.LoadConfig(strings.NewReader(yamlSrc))
	if err != nil {
		t.Fatalf("matrix 配置非法: %v", err)
	}
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: ev, DecisionProvider: dec})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// stubDec 透传固定决策。
type stubDec struct {
	dec kas.Decision
	err error
}

func (s stubDec) Decide(ctx context.Context, in kas.DecisionInput) (kas.Decision, error) {
	return s.dec, s.err
}

func (stubDec) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

func answerDecision() stubDec {
	return stubDec{dec: kas.Decision{Action: kas.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9, AnswerText: "回答正文"}}
}

var _ = config.D
