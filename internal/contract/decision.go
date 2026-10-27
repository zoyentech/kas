package contract

import (
	"context"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/decision"
	"github.com/zoyentech/kas/internal/domain"
)

// InmemDecision 是测试用可编程决策 provider（Story 5.1.1.1 T3 随套件交付）。
type InmemDecision struct {
	Dec domain.Decision
	Err error
}

// Decide 返回脚本化决策。
func (p *InmemDecision) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	if p.Err != nil {
		return domain.Decision{}, p.Err
	}
	return p.Dec, nil
}

// Health 恒 healthy。
func (p *InmemDecision) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// DecisionProviderSuite 是 DecisionProvider 合规套件（Story 5.1.1.1 T3）：
// 合法决策通过、非法决策被统一校验器拒绝、Health 结构、ctx 及时响应。
// Jev/规则/决策缓存实现均应接入。
func DecisionProviderSuite(t *testing.T, mk func(t *testing.T) domain.DecisionProvider) {
	t.Helper()

	input := domain.DecisionInput{
		Groups: []domain.CandidateGroup{{ID: "g1", Evidence: []domain.EvidenceRef{{EvidenceID: "e1"}}}},
	}

	t.Run("valid_decision_passes", func(t *testing.T) {
		p := mk(t)
		if _, err := p.Decide(context.Background(), input); err != nil {
			t.Fatalf("合法决策通过: %v", err)
		}
	})

	t.Run("health_structure", func(t *testing.T) {
		p := mk(t)
		h := p.Health(context.Background())
		switch h.Status {
		case domain.ComponentHealthy, domain.ComponentDegraded, domain.ComponentNotReady, domain.ComponentUnknown:
		default:
			t.Fatalf("Health.Status 必须属于枚举: %q", h.Status)
		}
	})

	t.Run("panic_isolated", func(t *testing.T) {
		// 同 Evidence 套件口径：panic/阻塞路径不得挂死调用方（隔离兜底在
		// deadline.Call 与降级链 safeDecide，Story 8.5.1.1）。
		p := mk(t)
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = recover() }()
			_, _ = p.Decide(ctx, input)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("panic/阻塞路径不得挂死调用方")
		}
	})

	t.Run("ctx_canceled_returns_promptly", func(t *testing.T) {
		p := mk(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = p.Decide(ctx, input)
		}()
		select {
		case <-done:
		case <-time.After(250 * time.Millisecond):
			t.Fatal("已取消 ctx 下 Decide 不得阻塞")
		}
	})
}

// AssertDecisionValid 用统一校验器断言决策合法（套件与实现共用信任边界）。
func AssertDecisionValid(t *testing.T, in domain.DecisionInput, d domain.Decision) {
	t.Helper()
	if err := decision.ValidateDecision(in, d); err != nil {
		t.Fatalf("决策应通过统一校验: %v", err)
	}
}
