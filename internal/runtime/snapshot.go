package runtime

import (
	"time"

	"github.com/zoyentech/kas/internal/config"
)

// Snapshot 是不可变运行时快照（Story 2.2.2.1 T1）：Config + 脱敏
// config_version + 构建时刻。字段私有、无 setter，仅暴露只读访问器；
// 同一 snapshot 内请求、trace、响应元数据与 cache key 使用同一版本。
type Snapshot struct {
	cfg     config.Config
	version string
	builtAt time.Time
}

// BuildSnapshot 执行 New 级同步校验（docs/design/03 §1 清单）并固化版本。
// 校验失败返回 ErrInvalidConfig 聚合错误；此函数是 New 与 Reload 共用的
// "构建候选快照"步骤（失败即弃，不影响 current）。
func BuildSnapshot(cfg config.Config) (*Snapshot, error) {
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	v, err := config.Version(cfg)
	if err != nil {
		return nil, err
	}
	return &Snapshot{cfg: cfg, version: v, builtAt: time.Now()}, nil
}

// Config 返回快照配置（值拷贝；Config 内切片仍共享底层数组，调用方不得修改）。
func (s *Snapshot) Config() config.Config { return s.cfg }

// Version 返回脱敏 config_version（SHA-256 前 12 hex）。
func (s *Snapshot) Version() string { return s.version }

// Budget 返回该快照的五阶段预算表。
func (s *Snapshot) Budget() config.TimeoutBudget { return s.cfg.Timeouts }

// BuiltAt 返回快照构建时刻（诊断用途，不参与版本计算）。
func (s *Snapshot) BuiltAt() time.Time { return s.builtAt }
