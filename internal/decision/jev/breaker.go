package jev

import (
	"fmt"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
)

// BreakerState 是熔断状态（kas_provider_breaker_state gauge 值口径：
// 0=closed 1=open 2=half_open）。
type BreakerState int32

// 熔断状态枚举（gauge 值即枚举值）。
const (
	BreakerClosed   BreakerState = 0
	BreakerOpen     BreakerState = 1
	BreakerHalfOpen BreakerState = 2
)

// String 返回 gauge/Health 展示名。
func (s BreakerState) String() string {
	switch s {
	case BreakerOpen:
		return "open"
	case BreakerHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}

// Breaker 是 Jev 级熔断器（Story 8.5.2.1，F-019）：closed →（连续失败达阈值）
// open（快速失败，不烧 rejudge 预算）→（冷却期满）half_open 单探测 →
// 成功回 closed / 失败重回 open。并发安全；状态迁移发 BreakerEvent。
type Breaker struct {
	mu       sync.Mutex
	state    BreakerState
	failures int
	openedAt time.Time
	probing  bool // half-open 下仅放行单探测
	cfg      BreakerConfig
	bus      *obs.Bus
	clock    func() time.Time
}

// BreakerConfig 是熔断参数（config.JevBreakerConfig 的运行形态）。
type BreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	Cooldown         time.Duration
}

// DefaultBreakerConfig 默认：开启、连续失败 5 次熔断、冷却 30s。
func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{Enabled: true, FailureThreshold: 5, Cooldown: 30 * time.Second}
}

// NewBreaker 构建熔断器；bus 可空（无事件）。
func NewBreaker(cfg BreakerConfig, bus *obs.Bus) *Breaker {
	return &Breaker{state: BreakerClosed, cfg: cfg, bus: bus, clock: time.Now}
}

// SetClock 注入时钟（测试）。
func (b *Breaker) SetClock(fn func() time.Time) { b.clock = fn }

// State 返回当前状态。
func (b *Breaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Allow 判定是否放行本次调用（放行后调用方必须以 Record 反馈结果）。
// 禁用（Enabled=false）时恒放行且不计数（防御：即使被直接构造也保持惰性）。
func (b *Breaker) Allow() error {
	if !b.cfg.Enabled {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case BreakerClosed:
		return nil
	case BreakerOpen:
		elapsed := b.clock().Sub(b.openedAt)
		if elapsed >= b.cfg.Cooldown {
			b.transitionLocked(BreakerHalfOpen)
			b.probing = true
			return nil // 冷却期满：放行单探测
		}
		return fmt.Errorf("%w: jev breaker open（快速失败，剩余冷却约 %s）",
			domain.ErrDecisionUnavailable, b.cfg.Cooldown-elapsed)
	case BreakerHalfOpen:
		if b.probing {
			return fmt.Errorf("%w: jev breaker half-open 探测进行中", domain.ErrDecisionUnavailable)
		}
		b.probing = true
		return nil
	}
	return nil
}

// Record 反馈一次调用结果并驱动状态迁移（禁用时零介入）。
func (b *Breaker) Record(success bool) {
	if !b.cfg.Enabled {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if success {
		if b.state != BreakerClosed {
			b.transitionLocked(BreakerClosed)
		}
		b.failures = 0
		b.probing = false
		return
	}
	b.probing = false
	switch b.state {
	case BreakerHalfOpen:
		b.transitionLocked(BreakerOpen) // 探测失败：重回 open 重新冷却
	case BreakerClosed:
		b.failures++
		if b.failures >= b.cfg.FailureThreshold {
			b.transitionLocked(BreakerOpen)
		}
	case BreakerOpen:
		// Allow 已拦截，正常不应到达；防御不计
	}
}

// transitionLocked 迁移并发事件（须持锁；事件在锁内同步分发，与总线语义一致）。
func (b *Breaker) transitionLocked(next BreakerState) {
	prev := b.state
	b.state = next
	if next == BreakerOpen {
		b.openedAt = b.clock()
	}
	if b.bus != nil {
		b.bus.Emit(obs.BreakerEvent{Provider: "jev", State: next.String(), PrevState: prev.String()})
	}
}
