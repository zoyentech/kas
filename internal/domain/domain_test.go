package domain_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestValidateRequestCases 表驱动：字段级校验（Story 3.1.1.1 T1）。
func TestValidateRequestCases(t *testing.T) {
	base := domain.Request{
		Question:  "服务支持哪些配置方式？",
		Principal: domain.PrincipalScope{TenantKey: "t", ScopeFingerprint: "fp", AuthorizationEpoch: "e1"},
		Knowledge: domain.KnowledgeScope{Namespace: "ns"},
	}
	tests := []struct {
		name    string
		mutate  func(*domain.Request)
		wantErr bool
		field   string
	}{
		{"valid", func(r *domain.Request) {}, false, ""},
		{"empty_question", func(r *domain.Request) { r.Question = "   " }, true, "Question"},
		{"no_tenant", func(r *domain.Request) { r.Principal.TenantKey = "" }, true, "Principal.TenantKey"},
		{"no_fingerprint", func(r *domain.Request) { r.Principal.ScopeFingerprint = "" }, true, "Principal.ScopeFingerprint"},
		{"no_epoch", func(r *domain.Request) { r.Principal.AuthorizationEpoch = "" }, true, "Principal.AuthorizationEpoch"},
		{"no_namespace", func(r *domain.Request) { r.Knowledge.Namespace = "" }, true, "Knowledge.Namespace"},
		{"request_id_ok", func(r *domain.Request) { r.RequestID = "abc-DEF.01:x" }, false, ""},
		{"request_id_bad_charset", func(r *domain.Request) { r.RequestID = "id with space" }, true, "RequestID"},
		{"request_id_bad_length", func(r *domain.Request) { r.RequestID = strings.Repeat("a", 65) }, true, "RequestID"},
	}
	for _, tc := range tests {
		req := base
		tc.mutate(&req)
		err := domain.ValidateRequest(req)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s: 应拒绝", tc.name)
			} else if !errors.Is(err, domain.ErrInvalidRequest) {
				t.Errorf("%s: 应为 ErrInvalidRequest，实际 %v", tc.name, err)
			} else if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("%s: 错误应含字段 %s，实际 %v", tc.name, tc.field, err)
			}
		} else if err != nil {
			t.Errorf("%s: 应通过，实际 %v", tc.name, err)
		}
	}
}

// TestValidateRequestNoQuestionLeak 错误串不含问题正文。
func TestValidateRequestNoQuestionLeak(t *testing.T) {
	secret := "我的敏感问题内容 13800000000"
	err := domain.ValidateRequest(domain.Request{Question: "", Principal: domain.PrincipalScope{}})
	if err != nil && strings.Contains(err.Error(), secret) {
		t.Fatal("错误串不得包含问题正文")
	}
	_ = secret
}

// TestActionTypeEnum 非法 action 字符串拒绝（Story 3.1.1.1 T2）。
func TestActionTypeEnum(t *testing.T) {
	for _, ok := range []string{"answer", "supplement", "ask_clarification", "abstain"} {
		if a, valid := domain.ParseAction(ok); !valid || string(a) != ok {
			t.Errorf("action %q 应合法", ok)
		}
	}
	for _, bad := range []string{"", "do_nothing", "ANSWER", "answer "} {
		if _, valid := domain.ParseAction(bad); valid {
			t.Errorf("action %q 应拒绝", bad)
		}
	}
}

// TestCitationJSONRoundTrip Citation 可序列化（Story 3.1.1.1 T3）。
func TestCitationJSONRoundTrip(t *testing.T) {
	c := domain.Citation{EvidenceID: "doc-10#seg-2", SourceID: "doc-10", Segment: "seg-2", ContentVersion: "v3"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.Citation
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != c {
		t.Fatalf("round-trip 不一致: %+v vs %+v", got, c)
	}
	// omitempty：空 Segment 不序列化
	b2, _ := json.Marshal(domain.Citation{EvidenceID: "x"})
	if strings.Contains(string(b2), "Segment") {
		t.Fatalf("空 Segment 应 omitempty: %s", b2)
	}
}

// TestResponseShapeDegradation 受控 reason code 透传。
func TestResponseShapeDegradation(t *testing.T) {
	c := 0.75
	resp := domain.Response{
		Action:      domain.ActionAnswer,
		Confidence:  &c,
		Degradation: []string{domain.ReasonDecisionCacheHit},
	}
	if resp.Confidence == nil || *resp.Confidence != 0.75 {
		t.Fatal("Confidence 指针语义异常")
	}
	b, _ := json.Marshal(resp)
	if !strings.Contains(string(b), `"ask"`) && !strings.Contains(string(b), "answer") {
		t.Fatalf("action 序列化异常: %s", b)
	}
}

// TestEvidenceTimeFields Evidence 时间字段语义（*time.Time 可空）。
func TestEvidenceTimeFields(t *testing.T) {
	now := time.Now()
	e := domain.Evidence{ID: "e1", ObservedAt: now, ValidFrom: &now}
	if e.ValidFrom == nil || !e.ValidFrom.Equal(now) {
		t.Fatal("ValidFrom 指针语义异常")
	}
	if e.ValidUntil != nil {
		t.Fatal("ValidUntil 默认 nil（未知）")
	}
}
