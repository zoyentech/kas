package testutil

import (
	"sync"
	"time"
)

// Clock 抽象时间源，供 deadline 引擎与缓存 TTL 测试注入 fake clock。
// 生产代码使用 SystemClock；测试使用 FakeClock 精确推进时间。
type Clock interface {
	Now() time.Time
}

// SystemClock 返回系统真实时间。
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now() }

// NewFakeClock 返回固定在 start 的假时钟（并发安全）。
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

// FakeClock 是可手动推进的时钟（并发安全）。
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now 返回当前假时间。
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance 将假时间前移 d 并返回新时间。
func (c *FakeClock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return c.now
}
