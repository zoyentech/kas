package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/obs"
	kasutil "github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { kasutil.VerifyTestMain(m) }

func cfg() config.MetricsConfig {
	return config.MetricsConfig{
		Enabled: true, Namespace: "kas",
		LabelAllowlist: []string{"operation", "outcome", "action", "stage", "provider",
			"result", "reason", "route", "component", "status", "deadline_stage", "error_class"},
	}
}

// TestMetricIncrements 事件→指标增量断言（Story 6.2.1.2 T2；
// docs/05 §2 Metrics 行）。
func TestMetricIncrements(t *testing.T) {
	c := New(cfg())
	meta := obs.Meta{TraceID: "t1", ConfigVersion: "cv1", At: time.Now()}

	c.Notify(obs.RequestEndEvent{Meta: meta, Outcome: "ok", Action: "answer", Duration: 100 * time.Millisecond})
	if got := testutil.ToFloat64(c.requestsTotal.WithLabelValues("answer", "ok", "answer")); got != 1 {
		t.Fatalf("requests_total = %v", got)
	}

	c.Notify(obs.StageEndEvent{Meta: meta, Stage: "first_round", Provider: "evidence", Outcome: "ok", Duration: 10 * time.Millisecond})
	c.Notify(obs.StageDeadlineEvent{Meta: meta, Stage: "rejudge", DeadlineStage: "rejudge"})
	c.Notify(obs.StageDeadlineEvent{Meta: meta, Stage: "root", DeadlineStage: "root"})

	c.Notify(obs.CacheEvent{Meta: meta, Result: "miss"})
	c.Notify(obs.CacheEvent{Meta: meta, Result: "hit"})
	c.Notify(obs.CacheEvictionEvent{Meta: meta, Reason: "capacity"})
	c.Notify(obs.FilterEvent{Meta: meta, Reason: "expired", Count: 3})

	c.Notify(obs.ProviderErrorEvent{Meta: meta, Provider: "decision", Operation: "decide", ErrorClass: "timeout"})
	c.Notify(obs.FallbackEvent{Meta: meta, Route: "deterministic_rules", Reason: "jev_unavailable"})
	c.Notify(obs.HealthEvent{Meta: meta, Component: "jev", Status: "degraded"})

	if got := testutil.ToFloat64(c.cacheTotal.WithLabelValues("hit")); got != 1 {
		t.Fatalf("cache hit = %v", got)
	}
	if got := testutil.ToFloat64(c.evidenceFiltered.WithLabelValues("expired")); got != 3 {
		t.Fatalf("filtered expired = %v", got)
	}
	if got := testutil.ToFloat64(c.decisionFallback.WithLabelValues("deterministic_rules", "jev_unavailable")); got != 1 {
		t.Fatalf("fallback = %v", got)
	}
	if got := testutil.ToFloat64(c.providerErrors.WithLabelValues("decision", "decide", "timeout")); got != 1 {
		t.Fatalf("provider_errors = %v", got)
	}
	if got := testutil.ToFloat64(c.stageDeadline.WithLabelValues("rejudge")); got != 1 {
		t.Fatalf("stage_deadline = %v", got)
	}
	if got := testutil.ToFloat64(c.deadlineTotal.WithLabelValues("root")); got != 1 {
		t.Fatalf("deadline_total = %v", got)
	}
	if got := testutil.ToFloat64(c.healthStatus.WithLabelValues("jev", "degraded")); got != 1 {
		t.Fatalf("health_status = %v", got)
	}
}

// TestRegisterLifecycle 注入→采集→注销全流程（Story 6.2.1.2 T1）。
func TestRegisterLifecycle(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	c1 := New(cfg())
	if err := c1.RegisterTo(reg); err != nil {
		t.Fatal(err)
	}
	c1.Notify(obs.CacheEvent{Meta: obs.Meta{At: time.Now()}, Result: "miss"})
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	if len(families) == 0 {
		t.Fatal("应有指标族")
	}
	c2 := New(cfg())
	if err := c2.RegisterTo(reg); err == nil {
		t.Fatal("同名冲突应报错")
	}
	c1.Unregister()
	if err := c2.RegisterTo(reg); err != nil {
		t.Fatalf("注销后应可注册: %v", err)
	}
	if err := New(cfg()).RegisterTo(nil); err != nil {
		t.Fatal(err)
	}
}

// TestLabelAllowlistDropsLabels allowlist 外的 label 从定义中移除。
func TestLabelAllowlistDropsLabels(t *testing.T) {
	restricted := cfg()
	restricted.LabelAllowlist = []string{"operation", "outcome", "result"}
	c := New(restricted)
	// stage label 被移除后 vec 无 label：直接以无参 WithLabelValues 计数成功
	//（有 label 的 vec 会因参数数量不符 panic，由 recover 捕获证明）
	func() {
		defer func() { _ = recover() }() // 无 label 时不应 panic
		c.stageDeadline.WithLabelValues()
	}()
	c.Notify(obs.StageDeadlineEvent{Meta: obs.Meta{At: time.Now()}, Stage: "rejudge", DeadlineStage: "rejudge"})
	if got := testutil.ToFloat64(c.stageDeadline.WithLabelValues()); got != 1 {
		t.Fatalf("无 label 计数 = %v", got)
	}
}

// TestNoSensitiveLabels 输出样本经敏感值扫描（docs/04 §5 禁令逐条）。
func TestNoSensitiveLabels(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	c := New(cfg())
	if err := c.RegisterTo(reg); err != nil {
		t.Fatal(err)
	}
	meta := obs.Meta{TraceID: "secret-trace-id-123", ConfigVersion: "cv-secret", At: time.Now()}
	c.Notify(obs.RequestEndEvent{Meta: meta, Outcome: "ok", Action: "answer"})
	c.Notify(obs.ProviderErrorEvent{Meta: meta, Provider: "decision", Operation: "decide", ErrorClass: "timeout"})
	c.Notify(obs.HealthEvent{Meta: meta, Component: "jev", Status: "healthy"})

	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if bad := CheckLabelValues(l.GetValue()); len(bad) > 0 {
					t.Errorf("指标 %s 出现禁入 label 值 %q", f.GetName(), l.GetValue())
				}
			}
		}
		if strings.Contains(f.GetName(), "model") {
			t.Errorf("模型版本不得作为指标: %s", f.GetName())
		}
	}
}

// TestCheckLabelValues 机械防护函数本身。
func TestCheckLabelValues(t *testing.T) {
	if bad := CheckLabelValues("question: 你好吗", "tenant-1", "trace-abc", "config_version:v2"); len(bad) != 4 {
		t.Fatalf("禁令清单应全部命中: %v", bad)
	}
	if bad := CheckLabelValues("hit", "miss", "expired", "jev"); len(bad) != 0 {
		t.Fatalf("受控枚举不应命中: %v", bad)
	}
}
