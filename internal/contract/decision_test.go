package contract_test

import (
	"context"
	"testing"

	"github.com/zoyentech/kas/internal/contract"
	"github.com/zoyentech/kas/internal/decision"
	"github.com/zoyentech/kas/internal/domain"
)

// inmem 决策 provider 接入契约套件（Story 5.1.1.1 T3 验收）。
func TestInmemDecisionContract(t *testing.T) {
	contract.DecisionProviderSuite(t, func(t *testing.T) domain.DecisionProvider {
		return &contract.InmemDecision{Dec: domain.Decision{
			Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9,
		}}
	})
}

// TestViolatingDecisionCaught 违约实现（越界目标组）被统一校验器抓住
// （套件的合法性信任边界）。
func TestViolatingDecisionCaught(t *testing.T) {
	p := &contract.InmemDecision{Dec: domain.Decision{
		Action: domain.ActionAnswer, TargetGroupIDs: []string{"gX"}, Confidence: 0.9,
	}}
	in := domain.DecisionInput{Groups: []domain.CandidateGroup{{ID: "g1"}}}
	d, err := p.Decide(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.ValidateDecision(in, d); err == nil {
		t.Fatal("越界目标组的决策必须被统一校验器拒绝")
	}
}
