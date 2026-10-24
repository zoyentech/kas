package orchestrator_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/orchestrator"
	"github.com/zoyentech/kas/internal/runtime"
)

// TestMixedConcurrencyBaseline 混合并发基线（Story 3.3.2.1 T2，F-012）：
// Answer/Reload 并发下无 panic、无版本串线；-race + goleak。
func TestMixedConcurrencyBaseline(t *testing.T) {
	cfg, err := config.LoadConfig(strings.NewReader(tinyYAML))
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := runtime.BuildSnapshot(cfg)
	mgr := runtime.NewManager(snap)
	oldV := snap.Version()

	ev := &fakeEvidence{result: domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "e1"}}}}
	dec := &fakeDecision{dec: domain.Decision{Action: domain.ActionAnswer}}
	e := orchestrator.NewEngine(mgr, obs.NewBus(), ev, dec)

	mutated := strings.Replace(tinyYAML, "top_k: 4", "top_k: 5", 1)
	mutSnap, err := runtime.BuildSnapshot(mustCfg(t, mutated))
	if err != nil {
		t.Fatal(err)
	}

	const n = 40
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := e.Answer(context.Background(), sampleRequest())
			if err != nil {
				t.Errorf("Answer: %v", err)
				return
			}
			if resp.ConfigVersion != oldV && resp.ConfigVersion != mutSnap.Version() {
				t.Errorf("版本串线: %s", resp.ConfigVersion)
			}
		}(i)
	}
	for i := 0; i < n/4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := runtime.Reload(context.Background(), mgr, strings.NewReader(mutated)); err != nil {
				t.Errorf("Reload: %v", err)
			}
		}(i)
	}
	for i := 0; i < n/4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bad := strings.Replace(tinyYAML, "total: 400ms", "total: 7ms", 1)
			if _, err := runtime.Reload(context.Background(), mgr, strings.NewReader(bad)); err == nil {
				t.Error("无效 Reload 意外成功")
			} else if !errors.Is(err, domain.ErrInvalidConfig) {
				t.Errorf("无效 Reload 分类错误: %v", err)
			}
		}()
	}
	wg.Wait()

	if v := mgr.Version(); v != oldV && v != mutSnap.Version() {
		t.Fatalf("最终版本异常: %s", v)
	}
}

func mustCfg(t *testing.T, src string) config.Config {
	t.Helper()
	c, err := config.LoadConfig(strings.NewReader(src))
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	return c
}

// TestConcurrentAnswerUnderLoadAnswerOrder 高并发下事件顺序仍与阶段顺序一致。
func TestConcurrentAnswerUnderLoad(t *testing.T) {
	ev := &fakeEvidence{result: domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "e1"}}}}
	dec := &fakeDecision{dec: domain.Decision{Action: domain.ActionAnswer}}
	rec := &recorder{}
	e := newEngine(t, ev, dec, rec)

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.Answer(context.Background(), sampleRequest()); err != nil {
				t.Errorf("Answer: %v", err)
			}
		}()
	}
	wg.Wait()

	// 每个请求 4 个 stage_end（preliminary/first_round/rejudge/reserve）
	stageEnds := rec.ofKind("stage_end")
	if len(stageEnds) != 4*30 {
		t.Fatalf("stage_end 数量 = %d, want %d", len(stageEnds), 4*30)
	}
	// 每个请求内顺序正确（按 TraceID 分组验证）
	byTrace := map[string][]string{}
	for _, evnt := range stageEnds {
		se := evnt.(obs.StageEndEvent)
		byTrace[se.TraceID] = append(byTrace[se.TraceID], se.Stage)
	}
	for traceID, stages := range byTrace {
		want := "preliminary,first_round,rejudge,reserve"
		if got := strings.Join(stages, ","); got != want {
			t.Fatalf("trace %s 阶段顺序 %s, want %s", traceID, got, want)
		}
	}
}

// TestDeadlineInvariantUnderLoad 所有阶段 deadline 不晚于 caller deadline（F-002）。
func TestDeadlineInvariantUnderLoad(t *testing.T) {
	ev := &fakeEvidence{result: domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "e1"}}}}
	e := newEngine(t, ev, nil, nil)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callerDeadline := time.Now().Add(150 * time.Millisecond) // 比 total 400ms 更早
			ctx, cancel := context.WithDeadline(context.Background(), callerDeadline)
			defer cancel()
			if _, err := e.Answer(ctx, sampleRequest()); err != nil {
				t.Errorf("Answer: %v", err)
			}
			// 所有 provider 收到的 ctx deadline ≤ caller deadline
			ev.mu.Lock()
			defer ev.mu.Unlock()
			for _, c := range ev.ctxs {
				if d, ok := c.Deadline(); ok && d.After(callerDeadline.Add(time.Millisecond)) {
					t.Errorf("阶段 deadline %v 晚于 caller deadline %v", d, callerDeadline)
					return
				}
			}
		}()
	}
	wg.Wait()
}
