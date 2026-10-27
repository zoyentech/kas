package balance_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/balance"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func bcfg(low, critical float64, interval time.Duration) config.BalanceConfig {
	return config.BalanceConfig{
		Enabled: true, Interval: config.D(interval),
		LowThreshold: low, CriticalThreshold: critical,
	}
}

// TestClassifyBoundaries 四态判定边界（Story 8.2.1.1：>low → normal；
// >critical → low；≤critical → critical）。
func TestClassifyBoundaries(t *testing.T) {
	cfg := bcfg(100, 20, time.Minute)
	tests := []struct {
		amount float64
		want   balance.State
	}{
		{1000, balance.StateNormal},
		{100.01, balance.StateNormal},
		{100, balance.StateLow}, // == low 归 low
		{20.01, balance.StateLow},
		{20, balance.StateCritical}, // == critical 归 critical
		{0, balance.StateCritical},
	}
	for _, tc := range tests {
		if got := balance.Classify(tc.amount, cfg); got != tc.want {
			t.Errorf("amount=%v: %s, want %s", tc.amount, got, tc.want)
		}
	}
	// gauge 值映射
	for st, want := range map[balance.State]float64{
		balance.StateNormal: 3, balance.StateLow: 2, balance.StateCritical: 1, balance.StateUnknown: 0,
	} {
		if got := balance.StateValue(st); got != want {
			t.Errorf("StateValue(%s)=%v, want %v", st, got, want)
		}
	}
}

// TestJevSourceContract fake 驱动：正常/5xx/畸形体/env 缺失（Story 8.2.2.1）。
func TestJevSourceContract(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		fake := testutil.NewJevFakeServer(testutil.JevScript{BalanceBody: `{"amount":42.5,"unit":"credit"}`})
		defer fake.Close()
		t.Setenv("KAS_BAL_EP", fake.URL()+"/balance")
		t.Setenv("KAS_BAL_KEY", "k")
		src, err := balance.NewJevSource("KAS_BAL_EP", "KAS_BAL_KEY")
		if err != nil {
			t.Fatal(err)
		}
		info, err := src.Balance(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if info.Amount != 42.5 || info.Unit != "credit" || info.Source != "jev" {
			t.Fatalf("info 异常: %+v", info)
		}
	})
	t.Run("http_500", func(t *testing.T) {
		fake := testutil.NewJevFakeServer(testutil.JevScript{BalanceStatus: 500})
		defer fake.Close()
		t.Setenv("KAS_BAL_EP", fake.URL()+"/balance")
		t.Setenv("KAS_BAL_KEY", "k")
		src, _ := balance.NewJevSource("KAS_BAL_EP", "KAS_BAL_KEY")
		if _, err := src.Balance(context.Background()); err == nil {
			t.Fatal("5xx 应报错")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		fake := testutil.NewJevFakeServer(testutil.JevScript{BalanceBody: `{oops`})
		defer fake.Close()
		t.Setenv("KAS_BAL_EP", fake.URL()+"/balance")
		t.Setenv("KAS_BAL_KEY", "k")
		src, _ := balance.NewJevSource("KAS_BAL_EP", "KAS_BAL_KEY")
		_, err := src.Balance(context.Background())
		if !errors.Is(err, domain.ErrProviderContract) {
			t.Fatalf("畸形体应契约错误: %v", err)
		}
	})
	t.Run("unknown_field", func(t *testing.T) {
		fake := testutil.NewJevFakeServer(testutil.JevScript{BalanceBody: `{"amount":1,"unit":"c","extra":2}`})
		defer fake.Close()
		t.Setenv("KAS_BAL_EP", fake.URL()+"/balance")
		t.Setenv("KAS_BAL_KEY", "k")
		src, _ := balance.NewJevSource("KAS_BAL_EP", "KAS_BAL_KEY")
		if _, err := src.Balance(context.Background()); !errors.Is(err, domain.ErrProviderContract) {
			t.Fatalf("未知字段应拒绝: %v", err)
		}
	})
	t.Run("env_missing", func(t *testing.T) {
		if _, err := balance.NewJevSource("KAS_BAL_NOPE", "KAS_BAL_NOPE2"); !errors.Is(err, domain.ErrInvalidConfig) {
			t.Fatalf("env 缺失应 ErrInvalidConfig: %v", err)
		} else if !strings.Contains(err.Error(), "KAS_BAL_NOPE") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("错误应含 env 名且不含 key 值: %v", err)
		}
	})
}

// fakeSource 可脚本化余额源。
type fakeSource struct {
	info balance.Info
	err  error
}

func (f *fakeSource) Balance(ctx context.Context) (balance.Info, error) {
	if f.err != nil {
		return balance.Info{}, f.err
	}
	return f.info, nil
}

// TestPollerStateTransitions 状态迁移矩阵 + unknown 保留上次值 + 事件对账
// （Story 8.2.2.2）。
func TestPollerStateTransitions(t *testing.T) {
	bus := obs.NewBus()
	var events []obs.BalanceEvent
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if be, ok := ev.(obs.BalanceEvent); ok {
			events = append(events, be)
		}
	}))
	src := &fakeSource{}
	p := balance.NewPoller(src, bcfg(100, 20, time.Minute), bus)
	fake := testutil.NewFakeClock(time.Now())
	p.SetClock(fake.Now)

	// normal
	src.info = balance.Info{Source: "jev", Amount: 500, Unit: "credit"}
	p.PollForTest(tB{})
	if _, st, _ := p.Snapshot(); st != balance.StateNormal {
		t.Fatalf("应 normal: %s", st)
	}
	// low
	src.info.Amount = 50
	p.PollForTest(tB{})
	if _, st, _ := p.Snapshot(); st != balance.StateLow {
		t.Fatalf("应 low: %s", st)
	}
	// critical
	src.info.Amount = 5
	p.PollForTest(tB{})
	if _, st, _ := p.Snapshot(); st != balance.StateCritical {
		t.Fatalf("应 critical: %s", st)
	}
	// unknown（查询失败）：保留上次成功值
	src.err = errors.New("upstream down")
	p.PollForTest(tB{})
	last, st, _ := p.Snapshot()
	if st != balance.StateUnknown {
		t.Fatalf("应 unknown: %s", st)
	}
	if last.Amount != 5 || last.Unit != "credit" {
		t.Fatalf("unknown 应保留上次成功值: %+v", last)
	}
	// 恢复 normal
	src.err = nil
	src.info.Amount = 800
	p.PollForTest(tB{})
	if _, st, _ := p.Snapshot(); st != balance.StateNormal {
		t.Fatalf("恢复应 normal: %s", st)
	}

	// 事件对账：每次 poll 一条，状态字段与快照一致
	if len(events) != 5 {
		t.Fatalf("事件数 = %d, want 5", len(events))
	}
	wantStates := []string{"normal", "low", "critical", "unknown", "normal"}
	for i, ev := range events {
		if ev.State != wantStates[i] {
			t.Fatalf("事件 %d 状态 = %s, want %s", i, ev.State, wantStates[i])
		}
	}
}

// TestPollerBackoff 失败退避（interval×2^(n-1)，封顶 5×）。
func TestPollerBackoff(t *testing.T) {
	src := &fakeSource{err: errors.New("down")}
	p := balance.NewPoller(src, bcfg(100, 20, 10*time.Second), nil)
	want := []time.Duration{10 * time.Second, 20 * time.Second, 40 * time.Second, 50 * time.Second, 50 * time.Second}
	for i, w := range want {
		p.PollForTest(tB{})
		if got := p.NextDelayForTest(); got != w {
			t.Fatalf("第 %d 次失败后 delay=%s, want %s", i+1, got, w)
		}
	}
}

// TestPollerRunLifecycle Run 阻塞轮询且 ctx 取消即停（goleak 由 TestMain 保证）。
func TestPollerRunLifecycle(t *testing.T) {
	src := &fakeSource{info: balance.Info{Source: "jev", Amount: 1000, Unit: "credit"}}
	p := balance.NewPoller(src, bcfg(100, 20, 50*time.Millisecond), nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Run(ctx)
	}()
	// 首查立即执行：短等后应已有 normal 快照
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, st, _ := p.Snapshot(); st == balance.StateNormal {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, st, _ := p.Snapshot(); st != balance.StateNormal {
		t.Fatal("Run 首查应立即执行")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消后 Run 应退出")
	}
}

// tB 适配 testB（PollForTest 参数）。
type tB struct{}

func (tB) Helper() {}

var _ = http.StatusOK
