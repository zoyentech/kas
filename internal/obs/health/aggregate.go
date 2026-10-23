// Package health 实现 Health 聚合与探测（F-005；docs/design/04 §4）：
// required 故障 → not_ready；optional Jev 故障且安全 fallback → degraded；
// 全部正常 → healthy。探测并发、各自短独立 timeout，不执行用户查询、
// 不访问 Evidence cache、不运行模型生成。
package health

import (
	"github.com/kas-runtime/kas/internal/domain"
)

// Input 是聚合输入（纯逻辑，Story 6.1.1.1 T1 判定矩阵）。
type Input struct {
	Components map[string]domain.ComponentHealth
	Required   []string
	Optional   []string
	// DegradedWhenOptionalUnavailable 为 false 时 optional 失败不影响总态。
	DegradedWhenOptionalUnavailable bool
}

// Aggregate 三态判定：任意 required 组件非 healthy → not_ready；
// optional 非 healthy 且开关开 → degraded；否则 healthy。
func Aggregate(in Input) domain.HealthReport {
	report := domain.HealthReport{Components: in.Components, CheckedAt: timeNow()}
	status := domain.StatusHealthy
	for _, name := range in.Required {
		if !isHealthy(in.Components[name]) {
			return finalize(domain.StatusNotReady, report)
		}
	}
	if in.DegradedWhenOptionalUnavailable {
		for _, name := range in.Optional {
			if !isHealthy(in.Components[name]) {
				status = domain.StatusDegraded
				break
			}
		}
	}
	return finalize(status, report)
}

func isHealthy(c domain.ComponentHealth) bool { return c.Status == domain.ComponentHealthy }

func finalize(s domain.OverallStatus, r domain.HealthReport) domain.HealthReport {
	r.Status = s
	return r
}
