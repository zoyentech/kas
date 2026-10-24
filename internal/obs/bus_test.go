package obs_test

import (
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestBusMultiSubscriberOrder 多订阅者按注册序收到全量事件，顺序与阶段
// 执行顺序一致（Story 3.3.1.2 T2）。
func TestBusMultiSubscriberOrder(t *testing.T) {
	bus := obs.NewBus()
	var orderA, orderB []string
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) { orderA = append(orderA, ev.Kind()) }))
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) { orderB = append(orderB, ev.Kind()) }))

	meta := obs.Meta{TraceID: "t1", ConfigVersion: "v1", At: time.Now()}
	bus.Emit(obs.StageEndEvent{Meta: meta, Stage: "preliminary", Outcome: "ok"})
	bus.Emit(obs.CacheEvent{Meta: meta, Result: "miss"})
	bus.Emit(obs.RequestEndEvent{Meta: meta, Outcome: "ok"})

	want := []string{"stage_end", "cache", "request_end"}
	if len(orderA) != 3 || orderA[0] != want[0] || orderA[1] != want[1] || orderA[2] != want[2] {
		t.Fatalf("订阅者 A 顺序异常: %v, want %v", orderA, want)
	}
	if len(orderB) != 3 {
		t.Fatalf("订阅者 B 应收到全量: %v", orderB)
	}
}

// TestBusPanicIsolation 单订阅者 panic 不影响主流程与其余订阅者。
func TestBusPanicIsolation(t *testing.T) {
	bus := obs.NewBus()
	var got []string
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) { panic("boom") }))
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) { got = append(got, ev.Kind()) }))

	bus.Emit(obs.HealthEvent{Meta: obs.Meta{At: time.Now()}, Component: "jev", Status: "degraded"})
	if len(got) != 1 || got[0] != "health" {
		t.Fatalf("panic 订阅者不应影响其余订阅者: %v", got)
	}
}

// TestBusNilSafety nil 事件与 nil 订阅者安全忽略。
func TestBusNilSafety(t *testing.T) {
	bus := obs.NewBus()
	bus.Subscribe(nil)
	bus.Emit(nil) // 不应 panic
	if bus.Len() != 0 {
		t.Fatalf("nil 订阅者不应注册: %d", bus.Len())
	}
}
