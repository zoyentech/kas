package cache

import "time"

// TTLDecision 计算写入 TTL（Story 4.3.2.1 T2；docs/design/04 §2）：
//
//	TTL = min(配置 TTL, 来源新鲜度 TTL, 最早推导/显式截止 − now)
//
// 三来源：configured（evidence.cache.ttl）、freshness（来源新鲜度，可零表
// 示不约束）、earliestDeadline（本批 Evidence 最早 ValidUntil/推导截止，
// 零值表示不约束）。≤0 返回 0（不写入，调用方计 bypass）。
func TTLDecision(configured, freshness time.Duration, earliestDeadline, now time.Time) time.Duration {
	ttl := configured
	if freshness > 0 && freshness < ttl {
		ttl = freshness
	}
	if !earliestDeadline.IsZero() {
		untilDeadline := earliestDeadline.Sub(now)
		if untilDeadline < ttl {
			ttl = untilDeadline
		}
	}
	if ttl <= 0 {
		return 0
	}
	return ttl
}

// EarliestDeadline 返回一批 Evidence 中最早的截止时刻（nil 视为无约束；
// 零值 time.Time 表示本批无任何截止信息，调用方按 fail-closed 处理后
// 不会走到写入路径）。
func EarliestDeadline(deadlines map[string]time.Time) time.Time {
	var earliest time.Time
	for _, d := range deadlines {
		if earliest.IsZero() || d.Before(earliest) {
			earliest = d
		}
	}
	return earliest
}
