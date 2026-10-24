package testutil_test

import (
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) {
	testutil.VerifyTestMain(m)
}

func TestFakeClockAdvances(t *testing.T) {
	start := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	fc := testutil.NewFakeClock(start)
	if !fc.Now().Equal(start) {
		t.Fatalf("初始时间 %v != %v", fc.Now(), start)
	}
	next := fc.Advance(90 * time.Second)
	if want := start.Add(90 * time.Second); !next.Equal(want) || !fc.Now().Equal(want) {
		t.Fatalf("推进后时间 %v != %v", fc.Now(), want)
	}
}

func TestCheckLeaksPassesWhenNoLeak(t *testing.T) {
	testutil.CheckLeaks(t) // 演示：无泄漏时静默通过
}
