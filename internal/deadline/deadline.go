package deadline

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/kas-runtime/kas/internal/domain"
)

// RootCause 是 root deadline 超时的原因（WithDeadlineCause 注入）。
type RootCause struct{ Total time.Duration }

func (c RootCause) Error() string {
	return fmt.Sprintf("kas: root deadline (total=%s) exceeded", c.Total)
}

// Root 是请求根 context 的包装。阶段 ctx 只能经 Root.Stage 派生——
// Stage 返回普通 context.Context，无法再调用 Stage，从类型上杜绝
// "从上一阶段短 deadline 派生下一阶段"（docs/design/04 §1 专门警告）。
// pending 计数跟踪未 cancel 的阶段，供测试断言全路径 cancel 保证。
type Root struct {
	ctx      context.Context
	deadline time.Time
	started  time.Time
	pending  atomic.Int32
}

// NewRoot 构造 root：deadline = min(caller deadline, started+total)。
// caller 无 deadline 或更晚时取 started+total；caller 更早不可延长
// （硬规则）。返回的 cancel 必须由调用方 defer 执行（全路径 cancel 保证）。
func NewRoot(parent context.Context, started time.Time, total time.Duration) (*Root, context.CancelFunc) {
	d := started.Add(total)
	if pd, ok := parent.Deadline(); ok && pd.Before(d) {
		d = pd
	}
	ctx, cancel := context.WithDeadlineCause(parent, d, RootCause{Total: total})
	return &Root{ctx: ctx, deadline: d, started: started}, cancel
}

// Ctx 返回 root context（只读用途；不要以其派生阶段，用 Stage）。
func (r *Root) Ctx() context.Context { return r.ctx }

// Deadline 返回 root deadline。
func (r *Root) Deadline() time.Time { return r.deadline }

// Started 返回请求开始时刻。
func (r *Root) Started() time.Time { return r.started }

// Remaining 返回 root 剩余预算（可为负）。
func (r *Root) Remaining(now time.Time) time.Duration { return r.deadline.Sub(now) }

// Expired 判定 root 是否已到期。
func (r *Root) Expired(now time.Time) bool { return !now.Before(r.deadline) }

// StageCause 是阶段 deadline 超时的原因。
type StageCause struct {
	Stage  string
	Budget time.Duration
}

func (c StageCause) Error() string {
	return fmt.Sprintf("kas: stage %q deadline (budget=%s) exceeded", c.Stage, c.Budget)
}

// Stage 从 root 派生阶段 ctx：deadline = min(root deadline, stageStart+budget)。
// 不接受上一阶段 ctx（签名即防护）；返回的 cancel 必须被执行（全路径
// cancel 保证；PendingStages 供测试断言无泄漏）。
func (r *Root) Stage(stage string, stageStart time.Time, budget time.Duration) (context.Context, context.CancelFunc) {
	d := stageStart.Add(budget)
	if r.deadline.Before(d) {
		d = r.deadline
	}
	r.pending.Add(1)
	ctx, cancel := context.WithDeadlineCause(r.ctx, d, StageCause{Stage: stage, Budget: budget})
	return ctx, func() {
		r.pending.Add(-1)
		cancel()
	}
}

// PendingStages 返回尚未 cancel 的阶段数（0 = 全路径 cancel 保证达成）。
func (r *Root) PendingStages() int { return int(r.pending.Load()) }

// AssertStageInvariant 运行期断言：阶段 deadline 不得晚于 root deadline。
// 供测试与防御性调用（违反即 panic-in-test；正常数学上不可能违反，
// 因为 Stage 已取 min）。
func (r *Root) AssertStageInvariant(stageCtx context.Context, stage string) {
	d, ok := stageCtx.Deadline()
	if !ok || d.After(r.deadline) {
		panic(fmt.Sprintf("kas/deadline: 阶段 %q deadline %v 违反 root deadline %v 的不变量", stage, d, r.deadline))
	}
}

// ExpiredErr 构造"阶段未启动即因 root 已到期"的哨兵错误（root 越界守卫）。
func ExpiredErr(stage string) error {
	return fmt.Errorf("%w: stage=%s not started: root deadline already exceeded",
		domain.ErrDeadlineExceeded, stage)
}
