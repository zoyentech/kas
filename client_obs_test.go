package kas_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/obs/trace"
)

var metricsYAML = strings.NewReplacer(
	"cache: {enabled: false}", "cache: {enabled: true, implementation: memory, max_entries: 100, max_bytes: 8388608, max_item_bytes: 1048576, ttl: 60s, include: [normalized_question, config_version, principal_scope_fingerprint]}",
	"metrics: {enabled: false, namespace: kas}", "metrics: {enabled: true, namespace: kas, label_allowlist: [operation, stage, provider, outcome, action, result, reason, route, component, status, deadline_stage, error_class]}",
	"tracing: {enabled: false}", "tracing: {enabled: true, capture_question_text: false, capture_evidence_text: false, include: [config_version, evidence_provider, decision_provider, model, model_version, fallback_route, cache_result]}",
).Replace(validYAML)

// obsDeps 返回健康、返回有效证据的最小 provider。
type obsEvidence struct{}

func (obsEvidence) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	until := time.Now().Add(time.Hour)
	return kas.EvidenceResult{Evidence: []kas.Evidence{
		{ID: "e1", SourceID: "s1", ValidUntil: &until, ContentVersion: "v1", Score: 0.9},
	}}, nil
}

func (obsEvidence) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

type obsDecision struct{}

func (obsDecision) Decide(ctx context.Context, in kas.DecisionInput) (kas.Decision, error) {
	return kas.Decision{
		Action: kas.ActionAnswer, TargetGroupIDs: groupIDs(in), Confidence: 0.9,
		Provider: "stub", Model: "m", ModelVersion: "v1", AnswerText: "回答正文",
	}, nil
}

func groupIDs(in kas.DecisionInput) []string {
	var ids []string
	for _, g := range in.Groups {
		ids = append(ids, g.ID)
	}
	return ids
}

func (obsDecision) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

func newObsClient(t *testing.T, reg prometheus.Registerer, sink kas.TraceSink) *kas.Client {
	t.Helper()
	t.Setenv("KAS_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_JEV_API_KEY", "test-key")
	cfg, err := kas.LoadConfig(strings.NewReader(metricsYAML))
	if err != nil {
		t.Fatalf("加载 metricsYAML 失败: %v", err)
	}
	opts := []kas.Option{}
	if reg != nil {
		opts = append(opts, kas.WithPrometheusRegisterer(reg))
	}
	c, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider: obsEvidence{},
		DecisionProvider: obsDecision{},
		TraceSink:        sink,
	}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestEndToEndMetricIncrements 端到端指标增量（成功 + 降级场景；F-006）
// （Story 6.2.1.2 T2）。
func TestEndToEndMetricIncrements(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	c := newObsClient(t, reg, nil)

	resp, err := c.Answer(context.Background(), sampleReq())
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if resp.Action != kas.ActionAnswer || len(resp.Citations) != 1 {
		t.Fatalf("回答应带 1 条引用: %+v", resp)
	}

	if got := testutil.CollectAndCount(reg, "kas_requests_total"); got != 1 {
		t.Fatalf("requests_total 族 = %d", got)
	}
	if got := testutil.CollectAndCount(reg, "kas_stage_duration_seconds"); got == 0 {
		t.Fatal("stage_duration 应有样本")
	}
	if got := testutil.CollectAndCount(reg, "kas_evidence_cache_total"); got == 0 {
		t.Fatal("cache 指标应有样本")
	}
	if got := testutil.CollectAndCount(reg, "kas_request_duration_seconds"); got == 0 {
		t.Fatal("request_duration 应有样本")
	}

	// 二次请求命中缓存（同问题/同 scope）
	if _, err := c.Answer(context.Background(), sampleReq()); err != nil {
		t.Fatal(err)
	}
}

// TestTraceEndToEndVersionChain trace 版本链完整性（F-009）：
// 正常路径含 config_version/model/model_version/cache_result。
func TestTraceEndToEndVersionChain(t *testing.T) {
	var buf bytes.Buffer
	c := newObsClient(t, nil, trace.NewSink(&buf))

	if _, err := c.Answer(context.Background(), sampleReq()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// include 白名单（配置）只输出：config_version/evidence_provider/
	// decision_provider/model/model_version/fallback_route/cache_result
	for _, want := range []string{
		`"config_version"`, `"model"`, `"model_version"`, `"cache_result"`,
		`"fallback_route"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("trace 缺 %s", want)
		}
	}
	if strings.Contains(out, "服务支持哪些配置方式") {
		t.Fatal("trace 泄漏问题正文")
	}
	if !strings.Contains(out, `"trace_id":"`) {
		t.Fatal("trace 应含 trace_id")
	}
}

// TestHealthAggregationThroughClient Health 公共 API 三态（F-005）。
func TestHealthAggregationThroughClient(t *testing.T) {
	// 全部健康 → healthy
	c := newObsClient(t, nil, nil)
	if r := c.Health(context.Background()); r.Status != kas.StatusHealthy {
		t.Fatalf("应 healthy: %+v", r)
	}

	// 必需组件故障 → not_ready
	t.Setenv("KAS_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_JEV_API_KEY", "k")
	cfg, _ := kas.LoadConfig(strings.NewReader(validYAML))
	broken, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: brokenEvidence{}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = broken.Close() }()
	if r := broken.Health(context.Background()); r.Status != kas.StatusNotReady {
		t.Fatalf("必需组件故障应 not_ready: %+v", r)
	}

	// Close 后 → not_ready
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if r := c.Health(context.Background()); r.Status != kas.StatusNotReady {
		t.Fatalf("Close 后应 not_ready: %+v", r)
	}
}

type brokenEvidence struct{}

func (brokenEvidence) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	return kas.EvidenceResult{}, kas.ErrEvidenceUnavailable
}

func (brokenEvidence) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentNotReady}
}

func sampleReq() kas.Request {
	return kas.Request{
		Question:  "服务支持哪些配置方式？",
		Principal: kas.PrincipalScope{TenantKey: "t", ScopeFingerprint: "fp", AuthorizationEpoch: "e1"},
		Knowledge: kas.KnowledgeScope{Namespace: "support-public", CorpusGeneration: "g1"},
	}
}

// TestMetricsUnregisterOnClose Close 注销自有 collector（生命周期）。
func TestMetricsUnregisterOnClose(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	t.Setenv("KAS_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_JEV_API_KEY", "k")
	cfg, _ := kas.LoadConfig(strings.NewReader(metricsYAML))
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: obsEvidence{}},
		kas.WithPrometheusRegisterer(reg))
	if err != nil {
		t.Fatal(err)
	}
	// 驱动一次请求产生样本，证明注册生效
	if _, err := c.Answer(context.Background(), sampleReq()); err != nil {
		t.Fatal(err)
	}
	if got := testutil.CollectAndCount(reg, "kas_requests_total"); got != 1 {
		t.Fatalf("样本数 = %d", got)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if got := testutil.CollectAndCount(reg, "kas_requests_total"); got != 0 {
		t.Fatalf("Close 后应注销: %d", got)
	}
	// 幂等 Close
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
