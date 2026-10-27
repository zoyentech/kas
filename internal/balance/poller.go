package balance

import (
	"context"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/obs"
)

// Poller 周期轮询余额（Story 8.2.2.2）：成功按 Classify 判四态；失败记
// unknown（保留上次成功值）并指数退避（interval×2^n，上限 5×interval，
// 连续失败封顶 4 档）；每次结果发 BalanceEvent（gauge Set 幂等、label 有界）。
// 并发安全；Run 阻塞至 ctx 取消（Close 停止）。
type Poller struct {
	src Source
	cfg config.BalanceConfig
	bus *obs.Bus
	now func() time.Time

	mu         sync.Mutex
	last       Info      // 最近一次成功结果（unknown 态保留展示）
	lastOK     time.Time // 最近一次成功时刻
	state      State     // 当前态（初始 unknown）
	stateAt    time.Time // 当前态进入时刻
	failStreak int       // 连续失败计数（成功清零）
}

// NewPoller 构建轮询器（未启动）。
func NewPoller(src Source, cfg config.BalanceConfig, bus *obs.Bus) *Poller {
	return &Poller{
		src: src, cfg: cfg, bus: bus, now: time.Now,
		state: StateUnknown, stateAt: time.Now(),
	}
}

// SetClock 注入时钟（测试）。
func (p *Poller) SetClock(now func() time.Time) { p.now = now }

// Run 阻塞轮询直至 ctx 取消：首查立即执行，随后按 interval（失败退避）循环。
func (p *Poller) Run(ctx context.Context) {
	delay := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		p.poll(ctx)
		delay = p.nextDelay()
	}
}

// poll 执行一次查询并发布事件。
func (p *Poller) poll(ctx context.Context) {
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	info, err := p.src.Balance(pctx)
	now := p.now()

	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.failStreak++
		if p.state != StateUnknown {
			p.state = StateUnknown
			p.stateAt = now
		}
		p.publishLocked()
		return
	}
	p.failStreak = 0
	p.last = info
	p.lastOK = now
	if next := Classify(info.Amount, p.cfg); next != p.state {
		p.state = next
		p.stateAt = now
	}
	p.publishLocked()
}

// publishLocked 发布 BalanceEvent（须持锁；bus 同步分发自带 panic 隔离）。
func (p *Poller) publishLocked() {
	if p.bus == nil {
		return
	}
	p.bus.Emit(obs.BalanceEvent{
		Source: p.last.Source,
		State:  string(p.state),
		Amount: p.last.Amount,
		Unit:   p.last.Unit,
	})
}

// nextDelay 计算下次轮询延迟（持锁读失败连击后释放）。
func (p *Poller) nextDelay() time.Duration {
	p.mu.Lock()
	streak := p.failStreak
	p.mu.Unlock()
	base := p.cfg.Interval.Std()
	if streak == 0 {
		return base
	}
	// 退避：×2^(streak-1) 全量翻倍后封顶 5×（streak≥4 达上限）
	backoff := base
	for i := 1; i < streak; i++ {
		backoff *= 2
	}
	if max := 5 * base; backoff > max {
		backoff = max
	}
	return backoff
}

// PollForTest 同步执行一次轮询（测试专用；生产路径走 Run）。
// 命名容忍：与包外测试约定（balance_test 包经此驱动确定性状态机）。
func (p *Poller) PollForTest(tb testB) {
	tb.Helper()
	p.poll(context.Background())
}

// NextDelayForTest 返回下次延迟（测试专用）。
func (p *Poller) NextDelayForTest() time.Duration { return p.nextDelay() }

// testB 是 testing.TB 的最小约束（避免 import testing 到生产文件）。
type testB interface {
	Helper()
}

// Snapshot 返回当前快照（state/进入时刻；unknown 时 last 为上次成功值）。
func (p *Poller) Snapshot() (last Info, state State, stateAt time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.last, p.state, p.stateAt
}
