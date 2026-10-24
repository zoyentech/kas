package evidence

import (
	"time"

	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence/perm"
	"github.com/zoyentech/kas/internal/evidence/validity"
)

// FilterConfig 是组合过滤配置（有效期 + 权限）。
type FilterConfig struct {
	Validity validity.Policy
	Perm     perm.Checker
}

// FilterOutcome 是组合过滤结果：仅保留"当前授权且在有效期内"的 Evidence
// （docs/design/01 §4 第 5~6 步）；补检结果同样经本函数重新过滤（第 8 步）。
type FilterOutcome struct {
	Kept      []domain.Evidence
	Removed   map[string]int       // expired/future/unknown_validity/unauthorized
	Deadlines map[string]time.Time // EvidenceID → 截止（供 TTL min）
}

// FilterAll 执行有效期 + 权限组合过滤（唯一过滤路径，不绕过）。
func FilterAll(items []domain.Evidence, now time.Time, scope domain.PrincipalScope, cfg FilterConfig) FilterOutcome {
	out := FilterOutcome{Removed: map[string]int{}, Deadlines: map[string]time.Time{}}
	v := validity.Filter(items, now, cfg.Validity)
	for reason, n := range v.Removed {
		out.Removed[reason] += n
	}
	for _, ev := range v.Kept {
		if cfg.Perm != nil && !cfg.Perm.Allowed(ev, scope) {
			out.Removed[perm.ReasonUnauthorized]++
			continue
		}
		out.Kept = append(out.Kept, ev)
		out.Deadlines[ev.ID] = v.Deadlines[ev.ID]
	}
	return out
}
