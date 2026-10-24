package health_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/obs/health"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestAggregateMatrix 判定矩阵（required/optional × 正常/故障 × 开关）
// （Story 6.1.1.1 T1，F-005）。
func TestAggregateMatrix(t *testing.T) {
	healthy := domain.ComponentHealth{Status: domain.ComponentHealthy}
	down := domain.ComponentHealth{Status: domain.ComponentNotReady}

	tests := []struct {
		name       string
		components map[string]domain.ComponentHealth
		required   []string
		optional   []string
		degraded   bool
		want       domain.OverallStatus
	}{
		{
			"all_healthy",
			map[string]domain.ComponentHealth{"evidence_provider": healthy, "jev": healthy},
			[]string{"evidence_provider"},
			[]string{"jev"},
			true, domain.StatusHealthy,
		},
		{
			"required_down",
			map[string]domain.ComponentHealth{"evidence_provider": down, "jev": healthy},
			[]string{"evidence_provider"},
			[]string{"jev"},
			true, domain.StatusNotReady,
		},
		{
			"optional_down_flag_on",
			map[string]domain.ComponentHealth{"evidence_provider": healthy, "jev": down},
			[]string{"evidence_provider"},
			[]string{"jev"},
			true, domain.StatusDegraded,
		},
		{
			"optional_down_flag_off",
			map[string]domain.ComponentHealth{"evidence_provider": healthy, "jev": down},
			[]string{"evidence_provider"},
			[]string{"jev"},
			false, domain.StatusHealthy,
		},
		{
			"required_missing_component",
			map[string]domain.ComponentHealth{"jev": healthy},
			[]string{"evidence_provider"},
			nil, true, domain.StatusNotReady,
		},
	}
	for _, tc := range tests {
		got := health.Aggregate(health.Input{
			Components: tc.components, Required: tc.required,
			Optional: tc.optional, DegradedWhenOptionalUnavailable: tc.degraded,
		})
		if got.Status != tc.want {
			t.Errorf("%s: status = %s, want %s", tc.name, got.Status, tc.want)
		}
	}
}

// blockingHealth 组件 Health 阻塞到 ctx.Done。
type blockingHealth struct{ calls int32 }

func (b *blockingHealth) Health(ctx context.Context) domain.ComponentHealth {
	b.calls++
	<-ctx.Done()
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

type fastHealth struct{ st domain.ComponentStatus }

func (f *fastHealth) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: f.st}
}

// fakeEvidence/fakeDecision 只用 Health 方法。
type evProvider struct{ h *blockingHealth }

func (evProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	return domain.EvidenceResult{}, nil
}
func (p evProvider) Health(ctx context.Context) domain.ComponentHealth { return p.h.Health(ctx) }

type decProvider struct{ h *fastHealth }

func (decProvider) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	return domain.Decision{}, nil
}
func (p decProvider) Health(ctx context.Context) domain.ComponentHealth { return p.h.Health(ctx) }

// TestProbeIndependentTimeout 慢组件不拖垮整体：Health 在 check_timeout+ε
// 内返回（Story 6.1.1.1 T2；docs/05 §2 Health 行）。
func TestProbeIndependentTimeout(t *testing.T) {
	block := &blockingHealth{}
	cfg := config.HealthConfig{
		RequiredComponents:                      []string{"evidence_provider"},
		OptionalComponents:                      []string{"jev"},
		CheckTimeout:                            config.D(50 * time.Millisecond),
		DegradedWhenOptionalProviderUnavailable: true,
	}
	start := time.Now()
	report := health.Probe(context.Background(), cfg, health.Deps{
		Evidence: evProvider{h: block},
		Decision: decProvider{h: &fastHealth{st: domain.ComponentHealthy}},
	}, nil)
	elapsed := time.Since(start)
	if elapsed > 250*time.Millisecond {
		t.Fatalf("Health 总耗时 %s 超过 check_timeout+ε", elapsed)
	}
	if report.Status != domain.StatusNotReady {
		t.Fatalf("必需组件阻塞超时应 not_ready: %s", report.Status)
	}
	if report.Components["evidence_provider"].Status != domain.ComponentNotReady {
		t.Fatal("超时组件应标记 not_ready")
	}
	if block.calls != 1 {
		t.Fatalf("探测计数 = %d", block.calls)
	}
}

// TestProbeHealthEvents 事件发布（component/status 枚举 label）。
func TestProbeHealthEvents(t *testing.T) {
	bus := obs.NewBus()
	var mu sync.Mutex
	var events []obs.HealthEvent
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if he, ok := ev.(obs.HealthEvent); ok {
			mu.Lock()
			events = append(events, he)
			mu.Unlock()
		}
	}))
	cfg := config.HealthConfig{
		RequiredComponents:                      []string{"evidence_provider"},
		CheckTimeout:                            config.D(time.Second),
		DegradedWhenOptionalProviderUnavailable: true,
	}
	_ = health.Probe(context.Background(), cfg, health.Deps{
		Evidence: evProvider{h: &blockingHealth{}}, // 阻塞但 timeout 1s 内会超时？
	}, bus)
	mu.Lock()
	defer mu.Unlock()
	if len(events) == 0 {
		t.Fatal("应发布 HealthEvent")
	}
	for _, e := range events {
		if e.Component == "" || e.Status == "" {
			t.Fatalf("事件字段不完整: %+v", e)
		}
	}
}

// TestProbeHostChecks 宿主注入检查点参与聚合。
func TestProbeHostChecks(t *testing.T) {
	cfg := config.HealthConfig{
		RequiredComponents:                      []string{"evidence_provider", "custom_dep"},
		CheckTimeout:                            config.D(time.Second),
		DegradedWhenOptionalProviderUnavailable: true,
	}
	report := health.Probe(context.Background(), cfg, health.Deps{
		Evidence: evProvider{h: &blockingHealth{}},
		Extra: domain.HealthChecks{
			"custom_dep": func(ctx context.Context) domain.ComponentHealth {
				return domain.ComponentHealth{Status: domain.ComponentNotReady, Detail: "down"}
			},
		},
	}, nil)
	if report.Status != domain.StatusNotReady {
		t.Fatalf("宿主 required 检查失败应 not_ready: %s", report.Status)
	}
	if report.Components["custom_dep"].Status != domain.ComponentNotReady {
		t.Fatal("宿主检查点结果缺失")
	}
}
