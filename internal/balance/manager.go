package balance

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/obs"
)

// Manager 管理多个命名余额源（v1.4 F-016 多源；参考 ai-support-platform
// 管理监控页的组件清单：jev/deepseek/aliyun(qwen 计费)…）：每源独立轮询器、
// 独立阈值/周期；支持运行时热更新（宿主管理后台换凭据/调阈值不重启进程）。
// 并发安全；Close 停止全部源。
type Manager struct {
	bus *obs.Bus

	mu      sync.Mutex
	entries map[string]*managedSource
	closed  bool
}

type managedSource struct {
	src    Source
	cfg    config.BalanceConfig // 合成后（全局默认 + 源覆盖）
	poller *Poller
	cancel context.CancelFunc
}

// NewManager 构建空管理器。
func NewManager(bus *obs.Bus) *Manager {
	return &Manager{bus: bus, entries: map[string]*managedSource{}}
}

// ResolveConfig 合成某源的生效配置（覆盖项回退全局默认）。
func ResolveConfig(global config.BalanceConfig, name string) config.BalanceConfig {
	out := global
	if override, ok := global.Sources[name]; ok {
		if override.Interval > 0 {
			out.Interval = override.Interval
		}
		if override.LowThreshold != nil {
			out.LowThreshold = *override.LowThreshold
		}
		if override.CriticalThreshold != nil {
			out.CriticalThreshold = *override.CriticalThreshold
		}
	}
	return out
}

// Upsert 注册或替换一个源（热更新入口）：先停旧轮询再起新轮询，中间态读旧快照
// 或 unknown；name 为受控标识（jev/deepseek/aliyun 或宿主自定义，作 gauge
// source label，禁止自由字符串增长——宿主应使用固定组件名）。
func (m *Manager) Upsert(name string, src Source, global config.BalanceConfig) {
	if src == nil || name == "" {
		return
	}
	cfg := ResolveConfig(global, name)
	poller := NewPoller(src, cfg, m.bus)
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		cancel()
		return
	}
	old := m.entries[name]
	m.entries[name] = &managedSource{src: src, cfg: cfg, poller: poller, cancel: cancel}
	m.mu.Unlock()

	if old != nil {
		old.cancel()
	}
	go poller.Run(ctx)
}

// SetConfig 热更新某源阈值/周期（不替换 Source，凭据不变）。
// 未注册的源返回 false。
func (m *Manager) SetConfig(name string, global config.BalanceConfig) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[name]
	if !ok {
		return false
	}
	e.cfg = ResolveConfig(global, name)
	e.poller.SetConfig(e.cfg)
	return true
}

// Remove 移除并停止一个源。
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	e, ok := m.entries[name]
	if ok {
		delete(m.entries, name)
	}
	m.mu.Unlock()
	if ok {
		e.cancel()
	}
}

// Names 返回已注册源名（排序稳定）。
func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.entries))
	for name := range m.entries {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Snapshot 返回某源快照；未注册返回 ok=false。
func (m *Manager) Snapshot(name string) (Info, State, time.Time, bool) {
	m.mu.Lock()
	e, ok := m.entries[name]
	m.mu.Unlock()
	if !ok {
		return Info{}, StateUnknown, time.Time{}, false
	}
	last, state, stateAt := e.poller.Snapshot()
	return last, state, stateAt, true
}

// SnapshotAll 返回全部源快照（供 Health 信息组件）。
func (m *Manager) SnapshotAll() map[string]StateInfo {
	m.mu.Lock()
	names := make([]string, 0, len(m.entries))
	entries := make(map[string]*managedSource, len(m.entries))
	for name, e := range m.entries {
		names = append(names, name)
		entries[name] = e
	}
	m.mu.Unlock()

	out := make(map[string]StateInfo, len(names))
	for _, name := range names {
		e := entries[name]
		last, state, stateAt := e.poller.Snapshot()
		out[name] = StateInfo{Last: last, State: state, StateAt: stateAt}
	}
	return out
}

// DefaultName 返回 Balance() 兼容口径的默认源：优先 jev，否则字典序首个。
func (m *Manager) DefaultName() (string, bool) {
	names := m.Names()
	if len(names) == 0 {
		return "", false
	}
	for _, n := range names {
		if n == "jev" {
			return n, true
		}
	}
	return names[0], true
}

// Close 停止全部源（幂等）。
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	cancels := make([]context.CancelFunc, 0, len(m.entries))
	for _, e := range m.entries {
		cancels = append(cancels, e.cancel)
	}
	m.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

// PollForTest 同步驱动某源一轮（测试专用）。
func (m *Manager) PollForTest(tb testB, name string) {
	tb.Helper()
	m.mu.Lock()
	e, ok := m.entries[name]
	m.mu.Unlock()
	if !ok {
		return
	}
	e.poller.PollForTest(tb)
}

// StateInfo 是某源对外快照（Health 消费）。
type StateInfo struct {
	Last    Info
	State   State
	StateAt time.Time
}
