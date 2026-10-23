package runtime

import "sync/atomic"

// Manager 持有 current snapshot 指针并以 atomic.Pointer 原子替换
// （Story 2.2.2.2 T1）。并发语义：
//   - 请求开始调用 Current() 读取一次并全程持有该指针（在途请求固定旧快照）
//   - Replace 瞬间并发 Current() 只见旧或新二者之一，绝无混合
type Manager struct {
	current atomic.Pointer[Snapshot]
}

// NewManager 以初始快照构建 Manager；initial 为 nil 时进入 not_ready 态
// （Config 为零值，调用方以 Version()=="" 识别）。
func NewManager(initial *Snapshot) *Manager {
	m := &Manager{}
	if initial != nil {
		m.current.Store(initial)
	}
	return m
}

// Current 返回当前快照；未加载配置时返回 nil。
func (m *Manager) Current() *Snapshot { return m.current.Load() }

// Replace 原子替换 current snapshot（发布顺序的最后一步）。
func (m *Manager) Replace(s *Snapshot) { m.current.Store(s) }

// Version 返回当前 config_version；未就绪返回空串。
func (m *Manager) Version() string {
	if s := m.current.Load(); s != nil {
		return s.Version()
	}
	return ""
}
