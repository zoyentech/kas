package matrix

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/domain"
)

// brokenHealthProvider：检索正常但 Health 报 not_ready。
type brokenHealthProvider struct{ inner *matrixEvidence }

func (p *brokenHealthProvider) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	return p.inner.Search(ctx, in)
}

func (p *brokenHealthProvider) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentNotReady, Detail: "db down"}
}

// 矩阵行 16：Health dependency 故障——required 为 not_ready；optional Jev
// 故障为 degraded。
func TestRowHealthFaults(t *testing.T) {
	// required（evidence）故障 → not_ready
	c := newClient(t, &brokenHealthProvider{inner: &matrixEvidence{}}, nil, nil)
	if r := c.Health(context.Background()); r.Status != kas.StatusNotReady {
		t.Fatalf("required 故障应 not_ready: %+v", r)
	}
	// 全部健康 → healthy
	c2 := newClient(t, &matrixEvidence{}, nil, nil)
	if r := c2.Health(context.Background()); r.Status != kas.StatusHealthy {
		t.Fatalf("应 healthy: %+v", r)
	}
}

// 矩阵行 19 + 17：并发 Answer/Health/Reload 无 snapshot 混用；Metrics 增量正确。
func TestRowMixedConcurrencyAndMetrics(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	yaml := func(src string) string {
		return strings.Replace(src, "metrics: {enabled: false, namespace: kas}",
			"metrics: {enabled: true, namespace: kas, label_allowlist: [operation, outcome, action, stage, provider, result, reason, route, component, status, deadline_stage, error_class]}", 1)
	}
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
	// metrics 启用需 Registerer
	t.Setenv("KAS_MTX_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_MTX_JEV_API_KEY", "k")
	cfg, err := kas.LoadConfig(strings.NewReader(yaml(baseYAML)))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider: ev,
		DecisionProvider: answerDecision(),
	}, kas.WithPrometheusRegisterer(reg))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	const n = 30
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Answer(context.Background(), req("混合并发问题")); err != nil {
				t.Errorf("Answer: %v", err)
			}
		}()
	}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Health(context.Background())
		}()
	}
	mutated := strings.Replace(yaml(baseYAML), "top_k: 4", "top_k: 5", 1)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Reload(context.Background(), strings.NewReader(mutated)); err != nil {
				t.Errorf("Reload: %v", err)
			}
		}()
	}
	wg.Wait()

	// Metrics 对账：30 次成功 answer 请求
	if got := testutil.CollectAndCount(reg, "kas_requests_total"); got != 1 {
		t.Fatalf("requests_total 应 1 个样本（answer/ok/answer）: %d", got)
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var requests float64
	for _, f := range families {
		if f.GetName() == "kas_requests_total" {
			for _, m := range f.GetMetric() {
				requests += m.GetCounter().GetValue()
			}
		}
	}
	if requests != float64(n) {
		t.Fatalf("requests_total = %v, want %d", requests, n)
	}
}
