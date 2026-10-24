package orchestrator_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/orchestrator"
)

// TestMapToPublicError 错误映射表驱动（Story 3.1.1.2 T2）：
// 任意内部错误映射到正确公共类别；公共文案固定，上游文本零外泄。
func TestMapToPublicError(t *testing.T) {
	upstream := errors.New("upstream 5xx body with user PII 张三 13800000000")
	tests := []struct {
		name     string
		err      error
		wantSent error
		wantMsg  string
	}{
		{"invalid_request", fmt.Errorf("wrap: %w", domain.ErrInvalidRequest), domain.ErrInvalidRequest, "kas: invalid request"},
		{"deadline", fmt.Errorf("stage=first_round cause=x: %w", domain.ErrDeadlineExceeded), domain.ErrDeadlineExceeded, "kas: deadline exceeded"},
		{"evidence_unavailable", fmt.Errorf("search failed: %w: %v", domain.ErrEvidenceUnavailable, upstream), domain.ErrEvidenceUnavailable, "kas: evidence unavailable"},
		{"decision_unavailable", fmt.Errorf("%w: %v", domain.ErrDecisionUnavailable, upstream), domain.ErrDecisionUnavailable, "kas: decision unavailable"},
		{"contract", fmt.Errorf("%w: action=do_nothing", domain.ErrProviderContract), domain.ErrProviderContract, "kas: provider contract violation"},
		{"closed", fmt.Errorf("%w", domain.ErrClosed), domain.ErrClosed, "kas: client closed"},
		{"ctx_deadline", fmt.Errorf("inner: %w", context.DeadlineExceeded), domain.ErrDeadlineExceeded, "kas: deadline exceeded"},
		{"ctx_canceled", fmt.Errorf("inner: %w", context.Canceled), domain.ErrDeadlineExceeded, "kas: deadline exceeded"},
		{"unknown_internal", fmt.Errorf("boom: %v", upstream), nil, "kas: internal error"},
	}
	for _, tc := range tests {
		got := orchestrator.MapToPublicError(tc.err)
		if tc.wantSent != nil && !errors.Is(got, tc.wantSent) {
			t.Errorf("%s: 哨兵判定失败: %v", tc.name, got)
		}
		if !strings.HasPrefix(got.Error(), tc.wantMsg) {
			t.Errorf("%s: 公共文案 = %q, want 前缀 %q", tc.name, got.Error(), tc.wantMsg)
		}
		if strings.Contains(got.Error(), "张三") || strings.Contains(got.Error(), "13800000000") {
			t.Errorf("%s: 上游文本外泄: %v", tc.name, got)
		}
	}
}

// TestReasonForError reason code 映射。
func TestReasonForError(t *testing.T) {
	if got := orchestrator.ReasonForError(fmt.Errorf("x: %w", domain.ErrDeadlineExceeded)); got != domain.ReasonDeadlineExceeded {
		t.Fatalf("deadline → %s", got)
	}
	if got := orchestrator.ReasonForError(nil); got != "" {
		t.Fatalf("nil → %q", got)
	}
}
