package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/kas-runtime/kas/internal/config"
)

// ReloadResult 是一次成功 Reload 的产物。
type ReloadResult struct {
	Snapshot *Snapshot
	Version  string
}

// Reload 执行五步发布流程（Story 2.2.2.2 T2；docs/design/06 §5）：
//
//  1. 解析   LoadConfig(r)            —— 严格 YAML + duration 校验
//  2. 校验   config.Validate          —— New 级启动校验（在 BuildSnapshot 内）
//  3. 准备   BuildSnapshot            —— 构建候选 snapshot（provider 准备由
//     调用方在 Replace 前注入，S3+ 扩展 prepare 钩子）
//  4. 版本   config.Version           ——（在 BuildSnapshot 内固化）
//  5. 替换   Manager.Replace          —— 原子替换
//
// 任一步失败：current pointer 保持原值，错误原样透出（ErrInvalidConfig 分类）。
// 在途请求持旧快照；新请求读新快照。
func Reload(ctx context.Context, m *Manager, r io.Reader) (ReloadResult, error) {
	if m == nil {
		return ReloadResult{}, errors.New("kas/runtime: manager 为 nil")
	}
	if err := ctx.Err(); err != nil {
		return ReloadResult{}, fmt.Errorf("kas: reload 取消: %w", err)
	}
	cfg, err := config.LoadConfig(r) // 1) 解析（含 duration 校验）
	if err != nil {
		return ReloadResult{}, err
	}
	snap, err := BuildSnapshot(cfg) // 2~4) 校验 + 准备 + 版本
	if err != nil {
		return ReloadResult{}, err
	}
	m.Replace(snap) // 5) 原子替换
	return ReloadResult{Snapshot: snap, Version: snap.Version()}, nil
}
