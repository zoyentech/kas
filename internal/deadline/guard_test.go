package deadline_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/deadline"
	"github.com/zoyentech/kas/internal/domain"
)

// TestGuardCallBlocksUntilDeadline 阻塞 provider（永不返回除非 ctx.Done）：
// Call 在阶段 deadline 内返回 ErrDeadlineExceeded（Story 3.2.1.2）。
func TestGuardCallBlocksUntilDeadline(t *testing.T) {
	stageStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cancelObserved := make(chan time.Time, 1)
	_, err := deadline.Call("first_round", ctx, stageStart, func(ctx context.Context) (string, error) {
		<-ctx.Done() // 合规 provider：响应 ctx.Done()
		cancelObserved <- time.Now()
		return "", ctx.Err()
	})
	if err == nil {
		t.Fatal("阻塞 provider 应以超时错误返回")
	}
	if !errors.Is(err, domain.ErrDeadlineExceeded) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("错误应为 DeadlineExceeded: %v", err)
	}
	// 阶段截止后 ≤50ms 内观察到取消（docs/05 §3 口径，测量误差上限）
	observedCancelAt := <-cancelObserved
	if lag := observedCancelAt.Sub(stageStart); lag > 100*time.Millisecond {
		t.Fatalf("provider 观察到取消延迟 %s 超过口径", lag)
	}
}

// TestGuardCallNeverWidensDeadline 包装层从不自建更宽 deadline：caller 已
// 取消则立即返回。
func TestGuardCallNeverWidensDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := deadline.Call("rejudge", ctx, time.Now(), func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) && !errors.Is(err, domain.ErrDeadlineExceeded) {
		t.Fatalf("已取消 ctx 应立即取消返回: %v", err)
	}
}

// TestGuardCallFastProvider 正常快速 provider 返回值透传。
func TestGuardCallFastProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := deadline.Call("rejudge", ctx, time.Now(), func(ctx context.Context) (int, error) {
		return 42, nil
	})
	if err != nil || got != 42 {
		t.Fatalf("got %d, err %v", got, err)
	}
}

// TestGuardRemaining 剩余预算计算（含调度容差）。
func TestGuardRemaining(t *testing.T) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	g := deadline.NewGuard("rejudge", ctx, start)
	if rem := g.Remaining(start); rem <= 0 || rem > time.Second+50*time.Millisecond {
		t.Fatalf("remaining = %s, want ~1s", rem)
	}
}
