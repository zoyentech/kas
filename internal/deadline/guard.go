package deadline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zoyentech/kas/internal/domain"
)

// Guard 包装 provider 调用（Story 3.2.1.2 T1）：注入阶段 ctx、透传取消、
// 将 ctx 错误映射为带阶段标注的 ErrDeadlineExceeded；从不自建更宽 deadline。
type Guard struct {
	Stage string
	Ctx   context.Context
	Start time.Time
}

// NewGuard 为阶段构建 provider 调用包装器。
func NewGuard(stage string, stageCtx context.Context, start time.Time) *Guard {
	return &Guard{Stage: stage, Ctx: stageCtx, Start: start}
}

// Call 在阶段 ctx 内执行 fn；ctx 取消/超时映射为 ErrDeadlineExceeded
// （错误串含阶段名与原因，不含上游响应体）。fn 在独立 goroutine 中运行：
// 即使 provider 不响应 ctx（违约），调用方也在阶段 deadline 内返回；
// 违约 goroutine 随宿主进程治理。
func Call[T any](stage string, ctx context.Context, start time.Time, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	done := make(chan callResult[T], 1)
	go func() {
		v, err := fn(ctx)
		done <- callResult[T]{v: v, err: err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			return zero, r.err
		}
		if err := ctx.Err(); err != nil {
			// 阶段已超时但 provider 已返回：仍按超时处理（不越过阶段预算采纳结果）。
			return zero, MapCtxErr(stage, ctx)
		}
		return r.v, nil
	case <-ctx.Done():
		return zero, MapCtxErr(stage, ctx)
	}
}

type callResult[T any] struct {
	v   T
	err error
}

// MapCtxErr 将 ctx 的取消/超时映射为带阶段与原因标注的哨兵错误。
func MapCtxErr(stage string, ctx context.Context) error {
	err := ctx.Err()
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		cause := context.Cause(ctx)
		return fmt.Errorf("%w: stage=%s cause=%v", domain.ErrDeadlineExceeded, stage, cause)
	}
	return err
}

// Remaining 返回阶段剩余预算（供 trace 记录剩余 deadline）。
func (g *Guard) Remaining(now time.Time) time.Duration {
	if d, ok := g.Ctx.Deadline(); ok {
		return d.Sub(now)
	}
	return 0
}
