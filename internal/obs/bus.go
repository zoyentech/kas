package obs

import (
	"log/slog"
	"sync"
)

// Subscriber 订阅内部事件（S6 的 metrics collector 与 trace sink 实现）。
type Subscriber interface {
	Notify(event Event)
}

// SubscriberFunc 函数适配器。
type SubscriberFunc func(event Event)

func (f SubscriberFunc) Notify(event Event) { f(event) }

// Bus 是同步分发的事件总线（Story 3.3.1.2 T2）：无额外 goroutine，分发在
// 请求路径内完成，避免泄漏面。单订阅者 panic 被隔离（recover + 安全日志），
// 不影响主流程与其余订阅者。
type Bus struct {
	mu   sync.RWMutex
	subs []Subscriber
	log  *slog.Logger
}

// NewBus 构建事件总线。
func NewBus() *Bus { return &Bus{log: slog.Default()} }

// Subscribe 注册订阅者；注册后立即开始收到全量后续事件。
func (b *Bus) Subscribe(s Subscriber) {
	if s == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, s)
}

// Emit 同步分发事件给全部订阅者，顺序与注册序一致；事件顺序与阶段执行
// 顺序一致（编排器在阶段节点同步 Emit）。
func (b *Bus) Emit(ev Event) {
	if ev == nil {
		return
	}
	b.mu.RLock()
	subs := b.subs
	b.mu.RUnlock()
	for _, s := range subs {
		b.safeNotify(s, ev)
	}
}

// safeNotify 隔离订阅者 panic（分发失败不影响请求结果）。
func (b *Bus) safeNotify(s Subscriber, ev Event) {
	defer func() {
		if r := recover(); r != nil {
			b.log.Error("kas/obs: 事件订阅者 panic 已隔离",
				"event_kind", ev.Kind(), "panic", r)
		}
	}()
	s.Notify(ev)
}

// Len 返回订阅者数量（测试用）。
func (b *Bus) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
