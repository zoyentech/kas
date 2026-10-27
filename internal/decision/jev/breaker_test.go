package jev_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/decision/jev"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/testutil"
)

func bcfg() jev.BreakerConfig {
	return jev.BreakerConfig{Enabled: true, FailureThreshold: 3, Cooldown: 30 * time.Second}
}

// TestBreakerTransitionMatrix 三态迁移矩阵（Story 8.5.2.1）。
func TestBreakerTransitionMatrix(t *testing.T) {
	fake := testutil.NewFakeClock(time.Now())
	bus := obs.NewBus()
	var events []obs.BreakerEvent
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if be, ok := ev.(obs.BreakerEvent); ok {
			events = append(events, be)
		}
	}))
	b := jev.NewBreaker(bcfg(), bus)
	b.SetClock(fake.Now)

	// closed：阈值内失败不 open
	b.Record(false)
	b.Record(false)
	if b.State() != jev.BreakerClosed {
		t.Fatalf("2/3 失败不应 open: %s", b.State())
	}
	// 第 3 次失败 → open（事件 closed→open）
	b.Record(false)
	if b.State() != jev.BreakerOpen {
		t.Fatalf("3 次连续失败应 open: %s", b.State())
	}
	// open 期：快速失败
	err := b.Allow()
	if !errors.Is(err, domain.ErrDecisionUnavailable) {
		t.Fatalf("open 期应快速失败: %v", err)
	}
	// 冷却未满仍 open
	fake.Advance(10 * time.Second)
	if err := b.Allow(); !errors.Is(err, domain.ErrDecisionUnavailable) || b.State() != jev.BreakerOpen {
		t.Fatalf("冷却未满应保持 open 快速失败: %v %s", err, b.State())
	}
	// 冷却期满 → allow 放行（half_open 单探测）
	fake.Advance(25 * time.Second)
	if err := b.Allow(); err != nil {
		t.Fatalf("冷却期满应放行探测: %v", err)
	}
	if b.State() != jev.BreakerHalfOpen {
		t.Fatalf("应处 half_open: %s", b.State())
	}
	// half_open 期间并发第二请求被拒（单探测）
	if err := b.Allow(); !errors.Is(err, domain.ErrDecisionUnavailable) {
		t.Fatalf("探测中应拒绝并发: %v", err)
	}
	// 探测成功 → closed（事件 half_open→closed）
	b.Record(true)
	if b.State() != jev.BreakerClosed {
		t.Fatalf("探测成功应回 closed: %s", b.State())
	}

	// 探测失败路径：重新 open
	b2 := jev.NewBreaker(bcfg(), nil)
	b2.SetClock(fake.Now)
	b2.Record(false)
	b2.Record(false)
	b2.Record(false)
	fake.Advance(31 * time.Second)
	if err := b2.Allow(); err != nil {
		t.Fatal("冷却期满应放行")
	}
	b2.Record(false)
	if b2.State() != jev.BreakerOpen {
		t.Fatalf("探测失败应重回 open: %s", b2.State())
	}

	// 事件序列核对
	if len(events) < 3 {
		t.Fatalf("迁移事件不足: %+v", events)
	}
	if events[0].State != "open" || events[0].PrevState != "closed" {
		t.Fatalf("首个迁移应为 closed→open: %+v", events[0])
	}
	if events[1].State != "half_open" || events[2].State != "closed" {
		t.Fatalf("后续迁移异常: %+v", events[1:])
	}
}

// TestBreakerDisabled 关闭时零介入。
func TestBreakerDisabled(t *testing.T) {
	cfg := bcfg()
	cfg.Enabled = false
	b := jev.NewBreaker(cfg, nil)
	for i := 0; i < 10; i++ {
		if err := b.Allow(); err != nil {
			t.Fatalf("禁用时不得拦截: %v", err)
		}
		b.Record(false)
	}
}

// TestBreakerOpenSkipsHTTP open 期间真实 Client 不发起 HTTP 调用
// （fake server 计数为 0）且快速返回（Story 8.5.2.1 验收口径）。
func TestBreakerOpenSkipsHTTP(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{Status: 500, Body: `{}`})
	defer fake.Close()
	t.Setenv("KAS_TEST_ENDPOINT", fake.URL())
	t.Setenv("KAS_TEST_API_KEY", "k")
	jc, err := jev.LoadConfig(jevCfg("KAS_TEST_ENDPOINT"), 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	c := jev.New(jc).WithBreaker(jev.BreakerConfig{Enabled: true, FailureThreshold: 3, Cooldown: time.Hour}, nil)
	in := decisionInput()
	for i := 0; i < 3; i++ {
		if _, err := c.Decide(context.Background(), in); err == nil {
			t.Fatal("5xx 应报错")
		}
	}
	if fake.Calls() != 3 {
		t.Fatalf("熔断前调用数 = %d, want 3", fake.Calls())
	}
	// 第 4 次：熔断 open，快速失败且零 HTTP 调用
	start := time.Now()
	if _, err := c.Decide(context.Background(), in); !errors.Is(err, domain.ErrDecisionUnavailable) {
		t.Fatalf("open 期应快速失败: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("open 期应远快于 rejudge 预算: %s", elapsed)
	}
	if fake.Calls() != 3 {
		t.Fatalf("open 期不得发起 HTTP: %d", fake.Calls())
	}
	// Health detail 可见
	if h := c.Health(context.Background()); h.Detail == "" {
		t.Fatal("Health detail 应含熔断状态")
	}
}
