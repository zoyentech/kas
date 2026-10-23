// Package perm 实现 Evidence 权限判定（Story 4.2.2.1；O-7 双模式）：
// 默认 PermissionTags 声明式映射，允许宿主注入判定函数。判定独立可注入，
// 宿主授权模型变化不影响 KAS 核心。
package perm

import "github.com/kas-runtime/kas/internal/domain"

// ReasonUnauthorized 是权限过滤原因（kas_evidence_filtered_total label）。
const ReasonUnauthorized = "unauthorized"

// Checker 是权限判定接口：单条 Evidence 对当前 principal scope 是否可用。
// 未授权由上层以 unauthorized 原因过滤。
type Checker interface {
	Allowed(ev domain.Evidence, scope domain.PrincipalScope) bool
}

// TagChecker 是声明式映射实现（O-7 默认模式）：scope fingerprint → 允许的
// PermissionTags 集合。语义：
//   - Evidence 无 PermissionTags（公开）→ 允许
//   - scope 在映射表中 → Evidence 的每个 tag 都须在允许集合内
//   - scope 不在映射表内且 Evidence 带权限要求 → 拒绝（fail-closed）
//
// 权限变化由宿主更换 ScopeFingerprint 或 AuthorizationEpoch（docs/design/06
// 权限 epoch 语义），自然更换 cache key 而非复用旧条目。
type TagChecker struct {
	AllowedTags map[string]map[string]bool // scopeFingerprint → tag set
}

// NewTagChecker 由 scope → tags 映射构建。
func NewTagChecker(allowed map[string][]string) *TagChecker {
	m := make(map[string]map[string]bool, len(allowed))
	for scope, tags := range allowed {
		set := make(map[string]bool, len(tags))
		for _, t := range tags {
			set[t] = true
		}
		m[scope] = set
	}
	return &TagChecker{AllowedTags: m}
}

// Allowed 执行声明式判定。
func (c *TagChecker) Allowed(ev domain.Evidence, scope domain.PrincipalScope) bool {
	if len(ev.PermissionTags) == 0 {
		return true
	}
	set, ok := c.AllowedTags[scope.ScopeFingerprint]
	if !ok {
		return false
	}
	for _, tag := range ev.PermissionTags {
		if !set[tag] {
			return false
		}
	}
	return true
}

// FuncChecker 是宿主注入判定函数模式（O-7 第二模式）。
type FuncChecker func(ev domain.Evidence, scope domain.PrincipalScope) bool

// Allowed 委托宿主函数。
func (f FuncChecker) Allowed(ev domain.Evidence, scope domain.PrincipalScope) bool {
	return f(ev, scope)
}

// AllowAll 放行全部（仅测试/无权限要求的部署形态）。
var AllowAll Checker = FuncChecker(func(ev domain.Evidence, scope domain.PrincipalScope) bool {
	return true
})
