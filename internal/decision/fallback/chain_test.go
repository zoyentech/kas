package fallback_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/decision/dcache"
	"github.com/zoyentech/kas/internal/decision/fallback"
	"github.com/zoyentech/kas/internal/decision/rules"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// stubDec 可脚本化 DecisionProvider。
type stubDec struct {
	dec domain.Decision
	err error
}

func (s stubDec) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	return s.dec, s.err
}

func (stubDec) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

func input() domain.DecisionInput {
	return domain.DecisionInput{
		TraceID: "t1", ConfigVersion: "v1",
		Groups: []domain.CandidateGroup{{ID: "g1", Evidence: []domain.EvidenceRef{{EvidenceID: "e1"}}}},
	}
}

func guard(t *testing.T) *rules.Engine {
	t.Helper()
	e, err := rules.New([]config.RuleConfig{
		{Name: "no_valid_evidence", When: config.CondValidEvidenceZero, Action: "abstain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// TestChainJevSuccess Jev 成功：直接返回（未降级）。
func TestChainJevSuccess(t *testing.T) {
	bus := obs.NewBus()
	jevOK := stubDec{dec: domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9, Provider: "jev"}}
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: jevOK},
		{Route: fallback.RouteRules, Provider: mustRules(t)},
	}, bus, guard(t))
	d, err := chain.Decide(context.Background(), input())
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "jev" || d.Degraded {
		t.Fatalf("Jev 成功不应降级: %+v", d)
	}
}

// TestChainFallbackOrder Jev 失败 → 规则命中 → 返回规则决策（F-008）。
func TestChainFallbackOrder(t *testing.T) {
	bus := obs.NewBus()
	var events []obs.FallbackEvent
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if fe, ok := ev.(obs.FallbackEvent); ok {
			events = append(events, fe)
		}
	}))
	jevFail := stubDec{err: errors.New("upstream 500")}
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: jevFail},
		{Route: fallback.RouteRules, Provider: mustRules(t)},
	}, bus, guard(t))
	// 无证据输入：Jev 失败后规则级 no_valid_evidence 命中 abstain
	in := domain.DecisionInput{TraceID: "t1", ConfigVersion: "v1", Groups: []domain.CandidateGroup{{ID: "g1"}}}
	d, err := chain.Decide(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "rules" || !d.Degraded || d.Action != domain.ActionAbstain {
		t.Fatalf("应降级到规则: %+v", d)
	}
	// 事件：jev 失败 → rules
	if len(events) != 1 || events[0].Route != fallback.RouteJev || events[0].Reason != domain.ReasonJevUnavailable {
		t.Fatalf("事件序列异常: %+v", events)
	}
}

// TestChainExhausted 全部耗尽 → abstain + chain_exhausted。
func TestChainExhausted(t *testing.T) {
	bus := obs.NewBus()
	noHit, err := rules.NewProvider(nil) // 无规则：永不命中
	if err != nil {
		t.Fatal(err)
	}
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: stubDec{err: errors.New("down")}},
		{Route: fallback.RouteRules, Provider: noHit},
		{Route: fallback.RouteCache, Provider: dcache.NewProvider(10, time.Minute)},
	}, bus, guard(t))
	in := input()
	d, err := chain.Decide(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != domain.ActionAbstain || d.ReasonCode != domain.ReasonDecisionExhausted || !d.Degraded {
		t.Fatalf("链耗尽应 abstain: %+v", d)
	}
}

// TestChainGuardrailOverridesSuccess Jev 成功但命中安全护栏（无据禁答）
// → 规则动作生效（可审计护栏）。
func TestChainGuardrailOverridesSuccess(t *testing.T) {
	bus := obs.NewBus()
	jevOK := stubDec{dec: domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9, Provider: "jev"}}
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: jevOK},
	}, bus, guard(t))
	// 无证据输入 → 护栏 no_valid_evidence 命中
	in := domain.DecisionInput{TraceID: "t", ConfigVersion: "v", Groups: []domain.CandidateGroup{{ID: "g1"}}}
	d, err := chain.Decide(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != domain.ActionAbstain || d.Provider != "rules" || !d.Degraded {
		t.Fatalf("护栏应覆盖 Jev answer: %+v", d)
	}
}

// TestChainBudgetGuard 剩余预算不足 → 立即安全终态，不启动下一级
// （Story 5.3.1.1 T3）。
func TestChainBudgetGuard(t *testing.T) {
	bus := obs.NewBus()
	slow := &slowDec{}
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: slow},
	}, bus, guard(t))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	d, err := chain.Decide(ctx, input())
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != domain.ActionAbstain {
		t.Fatalf("预算耗尽应 abstain: %+v", d)
	}
	if time.Since(start) > 300*time.Millisecond {
		t.Fatal("应在预算耗尽时立即返回")
	}
}

type slowDec struct{}

func (s *slowDec) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	<-ctx.Done()
	return domain.Decision{}, ctx.Err()
}

func (s *slowDec) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// TestChainThreeLevelEndToEnd 三级链端到端：Jev 失败 → 规则未命中 →
// 决策缓存命中（Story 5.3.2.1 T3）。
func TestChainThreeLevelEndToEnd(t *testing.T) {
	bus := obs.NewBus()
	dcp := dcache.NewProvider(10, time.Minute)
	// 预填缓存：同输入的有效 answer 决策
	in := input()
	dcp.Put(context.Background(), in, domain.Decision{
		Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9,
	})
	noHit, _ := rules.NewProvider(nil)
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: stubDec{err: errors.New("jev down")}},
		{Route: fallback.RouteRules, Provider: noHit},
		{Route: fallback.RouteCache, Provider: dcp},
	}, bus, guard(t))
	d, err := chain.Decide(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "decision_cache" || d.Action != domain.ActionAnswer || !d.Degraded {
		t.Fatalf("应命中决策缓存: %+v", d)
	}
}

// TestChainCacheWriterBackfill Jev 成功正向决策回填缓存；第二次 Jev 失败
// 后由缓存命中。
func TestChainCacheWriterBackfill(t *testing.T) {
	bus := obs.NewBus()
	dcp := dcache.NewProvider(10, time.Minute)
	flaky := &switchableDec{}
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: flaky},
		{Route: fallback.RouteCache, Provider: dcp},
	}, bus, guard(t))
	chain.CacheWriter = dcp
	in := input()

	// 第一次：Jev 成功（回填）
	flaky.set(domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9, Provider: "jev"})
	d, err := chain.Decide(context.Background(), in)
	if err != nil || d.Provider != "jev" {
		t.Fatalf("首次应 Jew 成功: %+v %v", d, err)
	}
	// 第二次：Jev 失败 → 缓存命中
	flaky.setErr(errors.New("down"))
	d, err = chain.Decide(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Provider != "decision_cache" || !d.Degraded {
		t.Fatalf("应回填后命中: %+v", d)
	}
}

type switchableDec struct {
	dec domain.Decision
	err error
}

func (s *switchableDec) set(d domain.Decision) { s.dec, s.err = d, nil }
func (s *switchableDec) setErr(err error)      { s.err = err }
func (s *switchableDec) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	return s.dec, s.err
}

func (s *switchableDec) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

func mustRules(t *testing.T) *rules.Provider {
	t.Helper()
	p, err := rules.NewProvider([]config.RuleConfig{
		{Name: "always", When: config.CondValidEvidenceZero, Action: "abstain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var _ = strings.Contains

// panicDec 恒 panic 的决策 provider（守卫用例）。
type panicDec struct{}

func (panicDec) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	panic("provider exploded")
}

func (panicDec) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// TestChainIsolatesStagePanic 链内单级 panic：该级按失败计、链继续下一级
// （Story 8.5.1.1）。
func TestChainIsolatesStagePanic(t *testing.T) {
	bus := obs.NewBus()
	chain := fallback.New([]fallback.Stage{
		{Route: fallback.RouteJev, Provider: panicDec{}},
		{Route: fallback.RouteRules, Provider: mustRules(t)},
	}, bus, guard(t))
	// 无证据输入：Jev 级 panic 后规则级命中 no_valid_evidence → abstain
	in := domain.DecisionInput{TraceID: "t", ConfigVersion: "v", Groups: []domain.CandidateGroup{{ID: "g1"}}}
	d, err := chain.Decide(context.Background(), in)
	if err != nil {
		t.Fatalf("panic 应被隔离且链继续: %v", err)
	}
	if d.Provider != "rules" || d.Action != domain.ActionAbstain {
		t.Fatalf("应降级到规则级: %+v", d)
	}
}
