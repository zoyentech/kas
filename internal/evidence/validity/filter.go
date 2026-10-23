// Package validity 实现知识有效期过滤（F-003）：expired/future/unknown
// fail-closed 三类规则与来源 max-age 推导截止时间。纯函数、无副作用；
// 事件发射由上层 evidence 过滤器聚合。
package validity

import (
	"time"

	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/domain"
)

// Reason 是过滤原因枚举（kas_evidence_filtered_total 的有界 label）。
const (
	ReasonExpired = "expired"
	ReasonFuture  = "future"
	ReasonUnknown = "unknown_validity"
)

// Policy 是有效期策略（自 config.ValidityConfig 派生的只读形态）。
type Policy struct {
	// ExcludeFutureDated 丢弃 ValidFrom 晚于当前时刻的 Evidence（默认 true）。
	ExcludeFutureDated bool
	// DefaultMaxAge 来源 → 批准的最大年龄（自 ObservedAt 推导截止）。
	DefaultMaxAge map[string]time.Duration
}

// FromConfig 由配置构建策略。
func FromConfig(cfg config.ValidityConfig) Policy {
	m := make(map[string]time.Duration, len(cfg.DefaultMaxAge))
	for k, v := range cfg.DefaultMaxAge {
		m[k] = v.Std()
	}
	return Policy{ExcludeFutureDated: cfg.ExcludeFutureDated, DefaultMaxAge: m}
}

// Outcome 是过滤结果。
type Outcome struct {
	Kept []domain.Evidence
	// Removed 各原因的移除计数（聚合，不含 Evidence 内容/ID）。
	Removed map[string]int
	// Deadlines 保留项的截止时刻（EvidenceID → 显式或推导 ValidUntil），
	// 供缓存 TTL 取 min（Story 4.2.1.1 T3 联动）。
	Deadlines map[string]time.Time
	// DerivedKept 经 max-age 推导（非显式 ValidUntil）的保留项数量。
	DerivedKept int
}

// Filter 过滤一批 Evidence（Story 4.2.1.1）：
//  1. future：ValidFrom != nil 且 now < ValidFrom（受 ExcludeFutureDated 开关）
//  2. 截止确定：显式 ValidUntil 优先；缺失时按来源 max-age 推导
//     （截止 = ObservedAt + max_age，推导值输出供 TTL 使用）
//  3. expired：now >= 截止（ValidUntil == now 视为过期）
//  4. unknown：无显式时间且无批准策略 → 一律丢弃（fail-closed；
//     v1.2 无 allow_unknown 放开路径，配置 fail_closed_on_unknown_validity
//     必须为 true，见 config 校验）
func Filter(items []domain.Evidence, now time.Time, p Policy) Outcome {
	out := Outcome{Removed: map[string]int{}, Deadlines: map[string]time.Time{}}
	for _, ev := range items {
		// 1) 未生效
		if p.ExcludeFutureDated && ev.ValidFrom != nil && now.Before(*ev.ValidFrom) {
			out.Removed[ReasonFuture]++
			continue
		}
		// 2) 截止确定
		deadline, hasDeadline := ev.ValidUntil, ev.ValidUntil != nil
		derived := false
		if !hasDeadline {
			if maxAge, ok := p.DefaultMaxAge[ev.SourceID]; ok {
				d := ev.ObservedAt.Add(maxAge)
				deadline, hasDeadline, derived = &d, true, true
			}
		}
		// 4) 有效期未知 → fail-closed
		if !hasDeadline {
			out.Removed[ReasonUnknown]++
			continue
		}
		// 3) 已过期（==now 视为过期）
		if !now.Before(*deadline) {
			out.Removed[ReasonExpired]++
			continue
		}
		out.Kept = append(out.Kept, ev)
		out.Deadlines[ev.ID] = *deadline
		if derived {
			out.DerivedKept++
		}
	}
	return out
}
