package decision_test

import (
	"errors"
	"math"
	"testing"

	"github.com/zoyentech/kas/internal/decision"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func input() domain.DecisionInput {
	return domain.DecisionInput{
		Groups: []domain.CandidateGroup{{ID: "g1"}, {ID: "g2"}},
	}
}

// TestValidateDecision 统一响应校验器三查（Story 5.1.1.1 T2；
// docs/05 §2 非法响应行）。
func TestValidateDecision(t *testing.T) {
	valid := domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.8}
	if err := decision.ValidateDecision(input(), valid); err != nil {
		t.Fatalf("合法决策被拒: %v", err)
	}

	tests := []struct {
		name string
		mut  func(*domain.Decision)
		want string // 错误应含的片段
	}{
		{"illegal_action", func(d *domain.Decision) { d.Action = "do_nothing" }, "action"},
		{"empty_action", func(d *domain.Decision) { d.Action = "" }, "action"},
		{"unknown_group", func(d *domain.Decision) { d.TargetGroupIDs = []string{"gX"} }, "不属于输入候选组"},
		{"nan_confidence", func(d *domain.Decision) { d.Confidence = math.NaN() }, "NaN"},
		{"inf_confidence", func(d *domain.Decision) { d.Confidence = math.Inf(1) }, "Inf"},
		{"confidence_above_1", func(d *domain.Decision) { d.Confidence = 1.5 }, "[0,1]"},
		{"confidence_below_0", func(d *domain.Decision) { d.Confidence = -0.1 }, "[0,1]"},
	}
	for _, tc := range tests {
		d := valid
		tc.mut(&d)
		err := decision.ValidateDecision(input(), d)
		if err == nil {
			t.Errorf("%s: 应拒绝", tc.name)
			continue
		}
		if !errors.Is(err, domain.ErrProviderContract) {
			t.Errorf("%s: 应分类 ErrProviderContract，实际 %v", tc.name, err)
		}
		if !contains(err.Error(), tc.want) {
			t.Errorf("%s: 错误应含 %q，实际 %v", tc.name, tc.want, err)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestFilterAttributes 白名单过滤（多余字段被剔除）。
func TestFilterAttributes(t *testing.T) {
	groups := []domain.CandidateGroup{{
		ID:         "g1",
		Attributes: map[string]string{"tier": "premium", "secret_field": "x", "kb": "internal"},
	}}
	got := decision.FilterAttributes(groups, []string{"tier", "kb"})
	if len(got) != 1 {
		t.Fatal("组数不变")
	}
	if got[0].Attributes["tier"] != "premium" || got[0].Attributes["kb"] != "internal" {
		t.Fatalf("白名单字段应保留: %v", got[0].Attributes)
	}
	if _, ok := got[0].Attributes["secret_field"]; ok {
		t.Fatal("白名单外字段应剔除")
	}
	// 空 allowlist：全剔除
	got2 := decision.FilterAttributes(groups, nil)
	if len(got2[0].Attributes) != 0 {
		t.Fatalf("空 allowlist 应剔除全部属性: %v", got2[0].Attributes)
	}
}
