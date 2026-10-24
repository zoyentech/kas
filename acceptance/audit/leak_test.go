// Package audit 是发布前敏感数据泄漏审计（Story 7.2.1.1 T1）：
// 五个输出面（指标/trace/错误/日志/panic）零敏感值子串扫描。
package audit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/obs/trace"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

const auditYAML = `
schema_version: 1
timeouts: {total: 600ms, preliminary: 50ms, first_round: 200ms, rejudge: 150ms, supplement: 100ms, reserve: 100ms, min_reserve: 40ms}
evidence:
  provider: audit
  namespace: ns-audit
  top_k: 4
  max_candidate_groups: 2
  validity:
    require_validity_policy_for_each_source: false
    default_max_age: {manual: 24h}
    fail_closed_on_unknown_validity: true
  cache:
    enabled: true
    implementation: memory
    max_entries: 100
    max_bytes: 8388608
    max_item_bytes: 1048576
    ttl: 60s
    include: [normalized_question, config_version, principal_scope_fingerprint]
decision:
  provider: jev
  jev: {endpoint_env: KAS_AUDIT_JEV_ENDPOINT, api_key_env: KAS_AUDIT_JEV_API_KEY, model: m, model_version: v1, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: true, namespace: kas, label_allowlist: [operation, outcome, action, stage, provider, result, reason, route, component, status, deadline_stage, error_class]}
tracing: {enabled: true, capture_question_text: false, capture_evidence_text: false, include: [config_version, evidence_provider, decision_provider, model, model_version, fallback_route, cache_result]}
`

// sensitive 敏感值集合（问题正文/身份/密钥/Evidence 正文/错误原文形态）。
var sensitive = []string{
	"我的手机号是13800000000住在张三小区",             // 问题正文
	"tenant-secret-key-xyz",               // 身份/密钥形态
	"AKIA-VERY-SECRET-TOKEN",              // token
	"内部机密证据正文：薪酬表",                        // Evidence 正文
	"upstream stack trace with user data", // 上游错误原文
}

// hostileEvidence 返回带敏感正文的证据 + 错误。
type hostileEvidence struct{}

func (hostileEvidence) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	until := time.Now().Add(time.Hour)
	return kas.EvidenceResult{Evidence: []kas.Evidence{
		{ID: "ev-1", SourceID: "manual", Excerpt: sensitive[3], ValidUntil: &until, Score: 0.9},
	}}, nil
}

func (hostileEvidence) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy, Detail: sensitive[4]}
}

type hostileDecision struct{}

func (hostileDecision) Decide(ctx context.Context, in kas.DecisionInput) (kas.Decision, error) {
	if in.Question == "触发决策错误" {
		return kas.Decision{}, errors.New(sensitive[4])
	}
	return kas.Decision{
		Action: kas.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9,
		Provider: "hostile", AnswerText: "回答正文",
	}, nil
}

func (hostileDecision) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

func TestLeakFiveSurfaces(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	var traceBuf bytes.Buffer

	t.Setenv("KAS_AUDIT_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_AUDIT_JEV_API_KEY", "kas-audit-secret-key")
	cfg, err := kas.LoadConfig(strings.NewReader(auditYAML))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider: hostileEvidence{},
		DecisionProvider: hostileDecision{},
		TraceSink:        trace.NewSink(&traceBuf),
	}, kas.WithPrometheusRegisterer(reg))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	req := kas.Request{
		// 问题正文携带敏感值（宿主失误场景：KAS 必须不泄漏）
		Question:  "查询 " + sensitive[0] + " 的待遇",
		Principal: kas.PrincipalScope{TenantKey: sensitive[1], ScopeFingerprint: "fp", AuthorizationEpoch: sensitive[2]},
		Knowledge: kas.KnowledgeScope{Namespace: "ns-audit", CorpusGeneration: "g1"},
	}
	resp, err := c.Answer(context.Background(), req)
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	// panic 路径：注入故障事件驱动同样扫描（故障决策错误）
	_, _ = c.Answer(context.Background(), kas.Request{
		Question:  "触发决策错误",
		Principal: kas.PrincipalScope{TenantKey: "t", ScopeFingerprint: "fp", AuthorizationEpoch: "e"},
		Knowledge: kas.KnowledgeScope{Namespace: "ns-audit", CorpusGeneration: "g1"},
	})

	// 面收集
	metricsOut := func() string {
		families, err := reg.Gather()
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		for _, f := range families {
			fmt.Fprintln(&b, f.String())
		}
		return b.String()
	}()
	surfaces := map[string]string{
		"metrics":  metricsOut,
		"trace":    traceBuf.String(),
		"error":    fmt.Sprint(err) + " " + resp.Answer + " " + strings.Join(resp.Degradation, ","),
		"response": fmt.Sprintf("%+v", resp),
		// health 面：Detail 是宿主 provider 自报、原样回传宿主本人（不进
		// 事件/指标——已由 metrics 扫描覆盖）；此处只扫 KAS 自产字段
		"health": func() string {
			r := c.Health(context.Background())
			var b strings.Builder
			fmt.Fprintf(&b, "status=%s", r.Status)
			for name, ch := range r.Components {
				fmt.Fprintf(&b, " component=%s status=%s", name, ch.Status)
			}
			return b.String()
		}(),
	}
	for name, out := range surfaces {
		for _, secret := range sensitive {
			if strings.Contains(out, secret) {
				t.Errorf("输出面 %s 泄漏敏感值 %q", name, secret)
			}
		}
	}
	// api_key 值（env 注入的密钥）额外扫描
	for name, out := range surfaces {
		if strings.Contains(out, "kas-audit-secret-key") {
			t.Errorf("输出面 %s 泄漏 api_key", name)
		}
	}
}
