package contract

import (
	"context"
	"testing"
	"time"
)

// 本文件是最小演示套件（Story 1.2.2.2 T3），展示正式契约套件（S4/S5 的
// EvidenceProvider/DecisionProvider/EvidenceCache）将采用的模式：
//
//	func XSuite[T <接口约束>](t *testing.T, mk func(t *testing.T) T)
//
// 套件覆盖：正常路径、ctx 取消传播、超时语义。每个用例经工厂获得全新实例，
// 实现方一行接入自测。

// AdderContract 是演示接口约束：实现必须在 ctx 取消后返回 ctx.Err()。
type AdderContract interface {
	Add(ctx context.Context, a, b int) (int, error)
}

// RunDemoSuite 对满足 AdderContract 的实现运行演示契约套件。
func RunDemoSuite[T AdderContract](t *testing.T, mk func(t *testing.T) T) {
	t.Helper()

	t.Run("normal", func(t *testing.T) {
		got, err := mk(t).Add(context.Background(), 2, 3)
		if err != nil {
			t.Fatalf("Add 返回错误: %v", err)
		}
		if got != 5 {
			t.Fatalf("Add(2,3) = %d, want 5", got)
		}
	})

	t.Run("context_cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := mk(t).Add(ctx, 1, 1); err == nil {
			t.Fatal("已取消的 ctx 必须返回错误（provider 须响应 ctx.Done()）")
		}
	})

	t.Run("context_deadline", func(t *testing.T) {
		// 不变量：provider 不得在 deadline 之后才返回；deadline 内完成是合法的
		// （契约针对阻塞型 provider 强制 ctx.Err()，由正式套件逐接口细化）。
		const budget = 50 * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		defer cancel()
		start := time.Now()
		_, _ = mk(t).Add(ctx, 1, 1)
		if elapsed := time.Since(start); elapsed > budget {
			t.Fatalf("provider 在 deadline 之后 %.3fs 才返回", elapsed.Seconds())
		}
	})
}
