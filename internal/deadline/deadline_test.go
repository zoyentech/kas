package deadline_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kas-runtime/kas/internal/deadline"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestRootMinDeadlineThreeStates 三态：caller 更早/更晚/无 deadline
// （Story 3.2.1.1 T1；docs/04 §1 硬规则"caller 更早时不可延长"）。
func TestRootMinDeadlineThreeStates(t *testing.T) {
	now := time.Now()

	// 无 caller deadline：root = start+total
	r1, cancel1 := deadline.NewRoot(context.Background(), now, time.Second)
	defer cancel1()
	if got := r1.Deadline(); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("无 caller deadline: %v, want %v", got, now.Add(time.Second))
	}

	// caller 更早：不可延长
	earlier := now.Add(300 * time.Millisecond)
	pCtx, pCancel := context.WithDeadline(context.Background(), earlier)
	defer pCancel()
	r2, cancel2 := deadline.NewRoot(pCtx, now, time.Second)
	defer cancel2()
	if got := r2.Deadline(); !got.Equal(earlier) {
		t.Fatalf("caller 更早: %v, want %v（不可延长）", got, earlier)
	}

	// caller 更晚：取 start+total
	later := now.Add(5 * time.Second)
	qCtx, qCancel := context.WithDeadline(context.Background(), later)
	defer qCancel()
	r3, cancel3 := deadline.NewRoot(qCtx, now, time.Second)
	defer cancel3()
	if got := r3.Deadline(); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("caller 更晚: %v, want %v", got, now.Add(time.Second))
	}
}

// TestStageDerivedOnlyFromRoot 阶段只从 root 派生：deadline = min(root, start+budget)，
// 且从不晚于 root（F-002 不变量）。
func TestStageDerivedOnlyFromRoot(t *testing.T) {
	now := time.Now()
	r, cancel := deadline.NewRoot(context.Background(), now, time.Second)
	defer cancel()

	// 预算小于剩余：stage deadline = start+budget
	sCtx, sCancel := r.Stage("first_round", now, 300*time.Millisecond)
	defer sCancel()
	if d, _ := sCtx.Deadline(); !d.Equal(now.Add(300 * time.Millisecond)) {
		t.Fatalf("stage deadline = %v, want %v", d, now.Add(300*time.Millisecond))
	}
	r.AssertStageInvariant(sCtx, "first_round")

	// 预算大于 root 剩余：截断到 root deadline
	lCtx, lCancel := r.Stage("rejudge", now.Add(800*time.Millisecond), 500*time.Millisecond)
	defer lCancel()
	if d, _ := lCtx.Deadline(); !d.Equal(now.Add(time.Second)) {
		t.Fatalf("stage deadline 应截断到 root: %v, want %v", d, now.Add(time.Second))
	}
}

// TestStageFromStageIsCompileTimePrevented Stage 返回普通 ctx，无法再派生
// 下一阶段（编译期防护的回归说明：下行无法编译——仅以类型断言记录）。
func TestStageFromStageIsCompileTimePrevented(t *testing.T) {
	var ctx context.Context = nil
	// deadline.Root 才有 Stage 方法；普通 context.Context 没有。
	// 若尝试 r.Stage(...) 的返回值再调 Stage 将无法编译：
	if _, ok := interface{}(ctx).(interface {
		Stage(string, time.Time, time.Duration) (context.Context, context.CancelFunc)
	}); ok {
		t.Fatal("普通 ctx 不应携带 Stage 方法")
	}
}

// TestCancelAllPaths 全路径 cancel：正常/错误/提前返回路径 PendingStages==0；
// 故意不 cancel 的泄漏用例被计数器抓住（Story 3.2.1.1 T3）。
func TestCancelAllPaths(t *testing.T) {
	now := time.Now()

	// 正常路径：全部 cancel
	r, cancel := deadline.NewRoot(context.Background(), now, time.Second)
	defer cancel()
	_, c1 := r.Stage("a", now, 100*time.Millisecond)
	c1()
	_, c2 := r.Stage("b", now, 100*time.Millisecond)
	c2()
	if r.PendingStages() != 0 {
		t.Fatalf("正常路径应无 pending: %d", r.PendingStages())
	}

	// 泄漏用例：故意不 cancel → 计数 > 0（测试基础设施抓住）
	r2, cancel2 := deadline.NewRoot(context.Background(), now, time.Second)
	defer cancel2()
	_, _ = r2.Stage("leaky", now, time.Hour) // 故意泄漏：不调用 cancel
	if r2.PendingStages() == 0 {
		t.Fatal("故意不 cancel 的阶段必须被 PendingStages 抓住")
	}
}

// TestRootDeadlineCause root 超时错误含原因（WithDeadlineCause）。
func TestRootDeadlineCause(t *testing.T) {
	r, cancel := deadline.NewRoot(context.Background(), time.Now(), 30*time.Millisecond)
	defer cancel()
	<-r.Ctx().Done()
	err := deadline.MapCtxErr("root", r.Ctx())
	if !errors.Is(err, domain.ErrDeadlineExceeded) {
		t.Fatalf("应映射 ErrDeadlineExceeded: %v", err)
	}
	if !strings.Contains(err.Error(), "root") || !strings.Contains(err.Error(), "root deadline") {
		t.Fatalf("错误应含阶段与原因: %v", err)
	}
}

// TestStageDeadlineCause 阶段超时错误含阶段名与阶段原因。
func TestStageDeadlineCause(t *testing.T) {
	r, cancel := deadline.NewRoot(context.Background(), time.Now(), time.Second)
	defer cancel()
	sCtx, sCancel := r.Stage("rejudge", time.Now(), 20*time.Millisecond)
	defer sCancel()
	<-sCtx.Done()
	err := deadline.MapCtxErr("rejudge", sCtx)
	if !errors.Is(err, domain.ErrDeadlineExceeded) {
		t.Fatalf("应映射 ErrDeadlineExceeded: %v", err)
	}
	if !strings.Contains(err.Error(), "rejudge") || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("错误应含阶段名与阶段原因: %v", err)
	}
}

// TestCallerCancelPropagates caller 主动取消传播到 root 与阶段。
func TestCallerCancelPropagates(t *testing.T) {
	pCtx, pCancel := context.WithCancel(context.Background())
	r, cancelR := deadline.NewRoot(pCtx, time.Now(), time.Second)
	defer cancelR()
	sCtx, sCancel := r.Stage("first_round", time.Now(), time.Second)
	defer sCancel()

	pCancel()
	select {
	case <-sCtx.Done():
	default:
		t.Fatal("caller 取消必须传播到阶段 ctx")
	}
}

// TestRootAccessors Started/Remaining/Expired 访问器。
func TestRootAccessors(t *testing.T) {
	now := time.Now()
	r, cancel := deadline.NewRoot(context.Background(), now, time.Second)
	defer cancel()
	if !r.Started().Equal(now) {
		t.Fatalf("Started = %v", r.Started())
	}
	if rem := r.Remaining(now); rem != time.Second {
		t.Fatalf("Remaining = %s", rem)
	}
	if r.Expired(now) {
		t.Fatal("尚未到期")
	}
	if !r.Expired(now.Add(time.Second + 1)) {
		t.Fatal("应已到期")
	}
}

// TestExpiredErrExpired 构造与哨兵判定。
func TestExpiredErrExpired(t *testing.T) {
	err := deadline.ExpiredErr("supplement")
	if !errors.Is(err, domain.ErrDeadlineExceeded) {
		t.Fatalf("应为 ErrDeadlineExceeded: %v", err)
	}
	if !strings.Contains(err.Error(), "supplement") {
		t.Fatalf("应含阶段名: %v", err)
	}
}

// TestAssertStageInvariantPanic 违反不变量 panic（防御性）。
func TestAssertStageInvariantPanic(t *testing.T) {
	now := time.Now()
	r, cancel := deadline.NewRoot(context.Background(), now, time.Second)
	defer cancel()
	sCtx, sCancel := r.Stage("ok", now, time.Millisecond)
	defer sCancel()
	r.AssertStageInvariant(sCtx, "ok") // 正常不 panic
	caught := func() (p any) {
		defer func() { p = recover() }()
		r.AssertStageInvariant(context.Background(), "bad") // 无 deadline ctx
		return nil
	}()
	if caught == nil {
		t.Fatal("无 deadline 的 ctx 应触发断言 panic")
	}
}

// TestMapCtxErrNilAndPlain MapCtxErr nil ctx 与非取消错误路径。
func TestMapCtxErrNilAndPlain(t *testing.T) {
	if err := deadline.MapCtxErr("x", context.Background()); err != nil {
		t.Fatalf("未取消 ctx 应返回 nil: %v", err)
	}
	// 非取消类 ctx 错误不存在于标准 context；该分支为防御性
	g := deadline.NewGuard("x", context.Background(), time.Now())
	if rem := g.Remaining(time.Now()); rem != 0 {
		t.Fatalf("无 deadline 的 ctx remaining 应为 0: %s", rem)
	}
}

// TestGuardCallProviderError provider 自身错误原样透传。
func TestGuardCallProviderError(t *testing.T) {
	sentinel := errors.New("provider exploded")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := deadline.Call("rejudge", ctx, time.Now(), func(ctx context.Context) (int, error) {
		return 0, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("provider 错误应透传: %v", err)
	}
}
