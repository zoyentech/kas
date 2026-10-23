package validity_test

import (
	"testing"
	"time"

	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/evidence/validity"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func at(t time.Time) *time.Time { return &t }

var policy = validity.Policy{
	ExcludeFutureDated: true,
	DefaultMaxAge:      map[string]time.Duration{"manual": 24 * time.Hour},
}

// TestFilterExpiredFutureBoundary expired/future 边界 ±1ns 与 ==now 语义
// （Story 4.2.1.1 T1：ValidUntil==now 视为过期）。
func TestFilterExpiredFutureBoundary(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		from   *time.Time
		until  *time.Time
		kept   bool
		reason string
	}{
		{"valid_explicit", at(now.Add(-time.Hour)), at(now.Add(time.Hour)), true, ""},
		{"expired_by_1ns", at(now.Add(-2 * time.Hour)), at(now.Add(-time.Nanosecond)), false, validity.ReasonExpired},
		{"expired_exactly_now", at(now.Add(-2 * time.Hour)), at(now), false, validity.ReasonExpired},
		{"future_by_1ns", at(now.Add(time.Nanosecond)), at(now.Add(time.Hour)), false, validity.ReasonFuture},
		{"future_exactly_now", at(now), at(now.Add(time.Hour)), true, ""},
	}
	for _, tc := range tests {
		ev := domain.Evidence{ID: "e", ValidFrom: tc.from, ValidUntil: tc.until, ObservedAt: now}
		out := validity.Filter([]domain.Evidence{ev}, now, policy)
		if tc.kept && len(out.Kept) != 1 {
			t.Errorf("%s: 应保留", tc.name)
		}
		if !tc.kept && out.Removed[tc.reason] != 1 {
			t.Errorf("%s: 应以 %s 移除，实际 %v", tc.name, tc.reason, out.Removed)
		}
	}
}

// TestFilterUnknownFailClosed 有效期未知且无来源策略 → fail-closed
// （Story 4.2.1.1 T2；docs/05 §2"有效期未知且无来源策略 → fail-closed"）。
func TestFilterUnknownFailClosed(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		evs  []domain.Evidence
	}{
		{
			"no_times_no_policy",
			[]domain.Evidence{{ID: "e1", SourceID: "unknown-source", ObservedAt: now}},
		},
		{
			"only_from_no_policy",
			[]domain.Evidence{{ID: "e2", SourceID: "unknown-source", ValidFrom: at(now.Add(-time.Hour)), ObservedAt: now}},
		},
		{
			"zero_observed_no_policy",
			[]domain.Evidence{{ID: "e3", SourceID: "nope"}},
		},
	}
	for _, tc := range tests {
		out := validity.Filter(tc.evs, now, policy)
		if len(out.Kept) != 0 || out.Removed[validity.ReasonUnknown] != len(tc.evs) {
			t.Errorf("%s: 应 fail-closed，实际 kept=%d removed=%v", tc.name, len(out.Kept), out.Removed)
		}
	}
}

// TestFilterMaxAgeDerivation max-age 推导截止 = ObservedAt + max_age；
// 推导后仍走 expired 判定；推导值输出供 TTL（Story 4.2.1.1 T3）。
func TestFilterMaxAgeDerivation(t *testing.T) {
	now := time.Now()
	keep := domain.Evidence{ID: "k1", SourceID: "manual", ObservedAt: now.Add(-23 * time.Hour)}
	expired := domain.Evidence{ID: "x1", SourceID: "manual", ObservedAt: now.Add(-25 * time.Hour)}
	out := validity.Filter([]domain.Evidence{keep, expired}, now, policy)
	if len(out.Kept) != 1 || out.Kept[0].ID != "k1" {
		t.Fatalf("推导后判定异常: %+v", out.Kept)
	}
	if out.Removed[validity.ReasonExpired] != 1 {
		t.Fatalf("推导过期应记 expired: %v", out.Removed)
	}
	wantDeadline := keep.ObservedAt.Add(24 * time.Hour)
	if got, ok := out.Deadlines["k1"]; !ok || !got.Equal(wantDeadline) {
		t.Fatalf("推导截止 = %v, want %v", got, wantDeadline)
	}
	if out.DerivedKept != 1 {
		t.Fatalf("DerivedKept = %d", out.DerivedKept)
	}
}

// TestFilterFutureSwitch exclude_future_dated=false 时 future 不被过滤。
func TestFilterFutureSwitch(t *testing.T) {
	now := time.Now()
	p := validity.Policy{ExcludeFutureDated: false}
	ev := domain.Evidence{ID: "f1", ValidFrom: at(now.Add(time.Hour)), ValidUntil: at(now.Add(2 * time.Hour))}
	out := validity.Filter([]domain.Evidence{ev}, now, p)
	if len(out.Kept) != 1 {
		t.Fatalf("开关关闭时 future 应保留: %v", out.Removed)
	}
}

// TestFilterMixedAggregation 混合场景：各 reason 计数聚合正确。
func TestFilterMixedAggregation(t *testing.T) {
	now := time.Now()
	evs := []domain.Evidence{
		{ID: "ok1", SourceID: "manual", ObservedAt: now.Add(-time.Hour)},
		{ID: "ok2", ValidUntil: at(now.Add(time.Hour))},
		{ID: "exp1", ValidUntil: at(now.Add(-time.Minute))},
		{ID: "fut1", ValidFrom: at(now.Add(time.Minute)), ValidUntil: at(now.Add(time.Hour))},
		{ID: "unk1", SourceID: "no-policy"},
	}
	out := validity.Filter(evs, now, policy)
	if len(out.Kept) != 2 {
		t.Fatalf("kept = %d, want 2", len(out.Kept))
	}
	if out.Removed[validity.ReasonExpired] != 1 || out.Removed[validity.ReasonFuture] != 1 || out.Removed[validity.ReasonUnknown] != 1 {
		t.Fatalf("计数异常: %v", out.Removed)
	}
}
