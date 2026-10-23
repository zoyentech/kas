// Package decision 实现决策契约：统一响应校验器（F-007/F-010）、
// 确定性规则引擎（rules）、Jev adapter（jev）与降级链（fallback）。
// 编排器只依赖 domain.DecisionProvider 接口，Jev 私有协议只存在于
// jev 包（F-007 封装边界）。
package decision

import (
	"fmt"
	"math"

	"github.com/kas-runtime/kas/internal/domain"
)

// ValidateDecision 统一响应安全校验器（Story 5.1.1.1 T2）：
//  1. action 属于有限枚举
//  2. 每个 TargetGroupID 必须来自输入 Groups（越界拒绝）
//  3. Confidence 有限（拒绝 NaN/±Inf）且在 [0,1]（docs/03 §3）
//
// 违规分类 ErrProviderContract；错误信息为固定枚举文案 + 违规项，
// 不含上游响应原文。
func ValidateDecision(in domain.DecisionInput, d domain.Decision) error {
	if !domain.ValidAction(d.Action) {
		return fmt.Errorf("%w: action %q 不属于有限枚举 answer/supplement/ask_clarification/abstain", domain.ErrProviderContract, d.Action)
	}
	allowed := make(map[string]bool, len(in.Groups))
	for _, g := range in.Groups {
		allowed[g.ID] = true
	}
	for _, id := range d.TargetGroupIDs {
		if !allowed[id] {
			return fmt.Errorf("%w: target_group_id %q 不属于输入候选组（共 %d 组）", domain.ErrProviderContract, id, len(in.Groups))
		}
	}
	if math.IsNaN(d.Confidence) || math.IsInf(d.Confidence, 0) {
		return fmt.Errorf("%w: confidence 为 NaN/Inf", domain.ErrProviderContract)
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("%w: confidence %v 超出 [0,1]", domain.ErrProviderContract, d.Confidence)
	}
	return nil
}

// FilterAttributes 按配置白名单过滤 CandidateGroup.Attributes
// （Story 5.1.1.1 T1：多余字段被剔除；白名单外键一律移除，防高基数与
// 敏感信息渗漏）。allowlist 为空时返回空 map（最严格）。
func FilterAttributes(groups []domain.CandidateGroup, allowlist []string) []domain.CandidateGroup {
	if len(allowlist) == 0 {
		for i := range groups {
			groups[i].Attributes = nil
		}
		return groups
	}
	set := make(map[string]bool, len(allowlist))
	for _, k := range allowlist {
		set[k] = true
	}
	for i := range groups {
		if len(groups[i].Attributes) == 0 {
			continue
		}
		filtered := make(map[string]string, len(groups[i].Attributes))
		for k, v := range groups[i].Attributes {
			if set[k] {
				filtered[k] = v
			}
		}
		if len(filtered) == 0 {
			filtered = nil
		}
		groups[i].Attributes = filtered
	}
	return groups
}
