// Package balance 实现余额监测（v1.3 F-016；ADR-0003）：BalanceSource 契约
// 与四态判定、Jev 适配（O-2 协议未定，最小契约 + fake 先行）、后台轮询器。
// Prometheus 是拉模式——余额采集必须内建于 SDK；展示经 BalanceEvent →
// kas_balance_amount/kas_balance_state gauge。
package balance

import (
	"context"
	"time"

	"github.com/zoyentech/kas/internal/config"
)

// State 是余额四态（kas_balance_state gauge 值口径：3=normal 2=low 1=critical 0=unknown）。
type State string

// 余额四态枚举。
const (
	StateNormal   State = "normal"
	StateLow      State = "low"
	StateCritical State = "critical"
	StateUnknown  State = "unknown"
)

// StateValue 返回 gauge 数值。
func StateValue(s State) float64 {
	switch s {
	case StateNormal:
		return 3
	case StateLow:
		return 2
	case StateCritical:
		return 1
	default:
		return 0
	}
}

// Info 是一次余额查询结果。
type Info struct {
	Source    string
	Amount    float64
	Unit      string // 有界枚举（如 credit/cny），禁止自由字符串增长
	FetchedAt time.Time
}

// Source 是余额来源契约（宿主可注入任意计费源；O-7 类似的可注入设计）。
// 实现必须响应 ctx.Done()；错误不含 secret 与余额值原文。
type Source interface {
	Balance(ctx context.Context) (Info, error)
}

// Classify 四态判定（Story 8.2.1.1 边界口径：amount > low → normal；
// amount > critical → low；≤ critical → critical；查询失败由调用方记 unknown）。
func Classify(amount float64, cfg config.BalanceConfig) State {
	switch {
	case amount > cfg.LowThreshold:
		return StateNormal
	case amount > cfg.CriticalThreshold:
		return StateLow
	default:
		return StateCritical
	}
}

// WithName 返回重标 Source 的包装（同一计费源映射到组件视图：如 Qwen 模型
// 计费=阿里云账户现金余额，注册为 "qwen" 以便按组件分列展示；v1.4.1/F-016）。
type namedSource struct {
	name string
	src  Source
}

// WithName 构建重标源。
func WithName(name string, src Source) Source {
	return &namedSource{name: name, src: src}
}

func (n *namedSource) Balance(ctx context.Context) (Info, error) {
	info, err := n.src.Balance(ctx)
	if err != nil {
		return Info{}, err
	}
	info.Source = n.name
	return info, nil
}
