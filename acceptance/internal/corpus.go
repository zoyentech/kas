package internal

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/cache"
	"github.com/zoyentech/kas/internal/domain"
)

// CorpusProvider 按规范化问题返回该用例的语料（acceptance 语料装载）：
// 期望证据 + 干扰证据（含 forbidden/过期/未授权形态，按 tag 语义构造）。
type CorpusProvider struct {
	mu       sync.Mutex
	byDigest map[string][]domain.Evidence
	calls    int
	validAt  time.Time
}

// BuildCorpus 由用例集构建语料 provider。证据形态：
//   - expected：有效（ValidUntil = valid_at + 24h）
//   - forbidden：同样可被检索命中（供断言"不进入引用"）——形态为已过期或
//     带越权 tag，真实地经过滤路径被拒
//   - 干扰项：tag 干扰（低相关有效证据）
func BuildCorpus(cases []Case, validAt time.Time) *CorpusProvider {
	p := &CorpusProvider{byDigest: map[string][]domain.Evidence{}, validAt: validAt}
	for _, c := range cases {
		digest := cache.QuestionDigest(cache.Normalize(c.Question, cache.CaseASCII), cache.CaseASCII)
		var ev []domain.Evidence
		for i, id := range c.ExpectedEvidenceIDs {
			until := validAt.Add(24 * time.Hour)
			ev = append(ev, domain.Evidence{
				ID: id, SourceID: sourceOf(id), Title: "证据 " + id,
				ValidUntil: &until, ObservedAt: validAt.Add(-time.Hour),
				ContentVersion: "v1", Score: 0.95 - float64(i)*0.1,
				// 内部知识库证据带权限 tag（permission 场景：scope-internal 放行）
				PermissionTags: permissionTagsOf(id),
			})
		}
		for i, id := range c.ForbiddenEvidenceIDs {
			// forbidden 形态：已过期（真实走过滤路径被拒）
			expired := validAt.Add(-time.Duration(i+1) * time.Hour)
			ev = append(ev, domain.Evidence{
				ID: id, SourceID: sourceOf(id), Title: "过期证据 " + id,
				ValidUntil: &expired, ObservedAt: expired.Add(-time.Hour),
				ContentVersion: "v0", Score: 0.5,
				// 同名知识库证据带权限 tag（越权 scope 不得引用）
				PermissionTags: permissionTagsOf(id),
			})
		}
		if len(ev) == 0 && !c.MustAbstain && c.ExpectedAction == "answer" {
			// 理论不达（answer 用例必有期望证据）；防御
			ev = append(ev, domain.Evidence{ID: c.CaseID + "-fallback"})
		}
		if _, ok := p.byDigest[digest]; !ok {
			p.byDigest[digest] = ev
		} else {
			p.byDigest[digest] = append(p.byDigest[digest], ev...)
		}
	}
	return p
}

// permissionTagsOf：内部知识库证据（kb-internal-*）要求 kb:internal 权限。
func permissionTagsOf(evidenceID string) []string {
	if strings.HasPrefix(evidenceID, "kb-internal") {
		return []string{"kb:internal"}
	}
	return nil
}

func sourceOf(evidenceID string) string {
	if i := indexOfStr(evidenceID, "#"); i > 0 {
		return evidenceID[:i]
	}
	return "src-" + evidenceID
}

func indexOfStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// Search 按规范化问题返回语料。
func (p *CorpusProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	digest := cache.QuestionDigest(cache.Normalize(in.Question, cache.CaseASCII), cache.CaseASCII)
	out := append([]domain.Evidence(nil), p.byDigest[digest]...)
	if in.TopK > 0 && len(out) > in.TopK {
		out = out[:in.TopK]
	}
	return domain.EvidenceResult{Evidence: out}, nil
}

// Health 恒 healthy。
func (p *CorpusProvider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// Calls 返回调用计数。
func (p *CorpusProvider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// GoldDecision 是金标决策 stub：期望证据全部在候选中 → answer（引用全部
// 有效证据）；否则 abstain/clarify（按期望动作）。回答文本拼接期望事实，
// 使事实支持率可评。
type GoldDecision struct{}

// Decide 实现金标决策。
func (GoldDecision) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	// 全部候选证据（经 KAS 过滤链后仅剩有效的）
	present := map[string]bool{}
	for _, g := range in.Groups {
		for _, e := range g.Evidence {
			present[e.EvidenceID] = true
		}
	}
	if len(present) == 0 {
		return domain.Decision{Action: domain.ActionAbstain, Provider: "gold", ReasonCode: "no_valid_evidence"}, nil
	}
	var ids []string
	for _, g := range in.Groups {
		ids = append(ids, g.ID)
	}
	facts := FactsFromContext(ctx)
	return domain.Decision{
		Action:         domain.ActionAnswer,
		TargetGroupIDs: ids,
		Confidence:     0.9,
		Provider:       "gold",
		Model:          "goldenset-stub",
		ModelVersion:   "v1",
		AnswerText:     facts,
	}, nil
}

// Health 恒 healthy。
func (GoldDecision) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

type factsKey struct{}

// WithFacts 把该用例的期望事实文本注入 ctx（runner 使用）。
func WithFacts(ctx context.Context, facts string) context.Context {
	return context.WithValue(ctx, factsKey{}, facts)
}

// FactsFromContext 取期望事实文本。
func FactsFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(factsKey{}).(string); ok {
		return v
	}
	return ""
}

// JoinFacts 拼接事实（回答文本）。
func JoinFacts(facts []string) string {
	out := ""
	for i, f := range facts {
		if i > 0 {
			out += "；"
		}
		out += f
	}
	if out == "" {
		out = "根据知识库整理的回答。"
	}
	return out
}
