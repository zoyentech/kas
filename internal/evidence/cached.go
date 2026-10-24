package evidence

import (
	"context"
	"time"

	"github.com/zoyentech/kas/internal/cache"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence/perm"
	"github.com/zoyentech/kas/internal/evidence/validity"
	"github.com/zoyentech/kas/internal/obs"
)

// CachedProvider 是缓存读写路径（Story 4.3.2.2；docs/design/04 §2）：
// 包装任意 EvidenceProvider，实现 domain.EvidenceProvider。编排器
// first_round 与 supplement 阶段统一经本路径检索——命中复核、miss 回源、
// 过滤后按 TTL min 写入、补检结果重新过滤（docs/design/01 §4 第 5/8 步），
// 过滤路径唯一不绕过。
//
//	miss 全路径：miss → provider → 权限+有效期过滤 → 有效结果写入
//	hit 路径：  hit → 当前权限+当前时刻复核 → 数量充足直接返回；不足量回源
//	           （回源结果为准，旧缓存数据绝不泄漏进答案）
type CachedProvider struct {
	Inner domain.EvidenceProvider
	Cache domain.EvidenceCache // 可空 = 缓存禁用（bypass）
	Cfg   config.EvidenceConfig
	Perm  perm.Checker // 权限判定（可空 = 不做权限过滤）
	Bus   *obs.Bus
	Clock func() time.Time
}

// Checker 是权限判定接口别名（供装配方引用）。
type Checker = perm.Checker

// NewCachedProvider 构建缓存化 provider。
func NewCachedProvider(inner domain.EvidenceProvider, c domain.EvidenceCache, cfg config.EvidenceConfig, checker Checker, bus *obs.Bus) *CachedProvider {
	return &CachedProvider{
		Inner: inner,
		Cache: c,
		Cfg:   cfg,
		Perm:  checker,
		Bus:   bus,
		Clock: time.Now,
	}
}

func (p *CachedProvider) filterConfig() FilterConfig {
	return FilterConfig{
		Validity: validity.FromConfig(p.Cfg.Validity),
		Perm:     p.Perm,
	}
}

// Search 执行缓存读写编排（在阶段 ctx 内调用；provider 调用透传 ctx）。
func (p *CachedProvider) Search(ctx context.Context, q domain.EvidenceQuery) (domain.EvidenceResult, error) {
	now := p.Clock()
	meta := obs.Meta{TraceID: q.TraceID, ConfigVersion: q.ConfigVersion, At: now}
	fc := p.filterConfig()

	// —— 命中路径：先复核再信任 ——
	if p.Cfg.Cache.Enabled && p.Cache != nil {
		key := p.keyFor(q)
		res, hit, err := p.Cache.Get(ctx, key)
		switch {
		case err != nil:
			p.emitCache(meta, "error")
		case hit:
			fo := FilterAll(res.Evidence, now, q.Scope, fc)
			p.emitFilters(meta, fo)
			if len(fo.Kept) >= effectiveTopK(q) {
				p.emitCache(meta, "hit")
				return domain.EvidenceResult{Evidence: fo.Kept, FromCache: true}, nil
			}
			// 复核后不足量：以 miss 语义回源（docs/04 §2"过滤后不足量时可回源"）
			p.emitCache(meta, "miss")
		default:
			p.emitCache(meta, "miss")
		}
	} else {
		p.emitCache(meta, "bypass")
	}

	// —— miss 全路径：回源 → 过滤 ——
	res, err := p.Inner.Search(ctx, q)
	if err != nil {
		// provider error / timeout：一律不缓存，原样透出（禁写清单）
		return domain.EvidenceResult{}, err
	}
	fo := FilterAll(res.Evidence, now, q.Scope, fc)
	p.emitFilters(meta, fo)

	// —— 写入路径 ——
	// 禁写清单（Story 4.3.2.1 T3）：空结果/全部被滤（有效期未知已在过滤中
	// 丢弃）/provider error（上方已透出）不写；cache_empty_results 仅放开
	// 原始空结果的缓存语义边界。
	if p.Cfg.Cache.Enabled && p.Cache != nil {
		if len(fo.Kept) > 0 || p.Cfg.Cache.CacheEmptyResults {
			ttl := cache.TTLDecision(
				p.Cfg.Cache.TTL.Std(),
				0, // 来源新鲜度 TTL：v1.2 provider 元数据未提供（不约束）
				cache.EarliestDeadline(fo.Deadlines),
				now,
			)
			if ttl > 0 {
				if err := p.Cache.Put(ctx, p.keyFor(q), domain.EvidenceResult{Evidence: fo.Kept}, ttl); err != nil {
					p.emitCache(meta, "error")
				}
			}
		}
	}
	return domain.EvidenceResult{Evidence: fo.Kept}, nil
}

// Health 委托 inner provider。
func (p *CachedProvider) Health(ctx context.Context) domain.ComponentHealth {
	return p.Inner.Health(ctx)
}

// keyFor 组装七维 key（问题规范化 + 各维度摘要化在 cache 层完成）。
func (p *CachedProvider) keyFor(q domain.EvidenceQuery) domain.CacheKey {
	in := cache.KeyInput{
		Question:           cache.Normalize(q.Question, cache.CaseASCII),
		ConfigVersion:      q.ConfigVersion,
		ScopeFingerprint:   q.Scope.ScopeFingerprint,
		AuthorizationEpoch: q.Scope.AuthorizationEpoch,
		Namespace:          q.Namespace,
		CorpusGeneration:   q.CorpusGeneration,
		CasePolicy:         cache.CaseASCII,
		Policy:             p.Cfg,
	}
	parts := in.Parts()
	return domain.CacheKey{
		QuestionDigest:         parts.QuestionDigest,
		ConfigVersion:          parts.ConfigVersion,
		ScopeFingerprint:       parts.ScopeFingerprint,
		AuthorizationEpoch:     parts.AuthorizationEpoch,
		KnowledgeNamespace:     parts.Namespace,
		CorpusGeneration:       parts.CorpusGeneration,
		RetrievalPolicyVersion: parts.PolicyVersion,
	}
}

func effectiveTopK(q domain.EvidenceQuery) int {
	if q.TopK > 0 {
		return q.TopK
	}
	return 1
}

func (p *CachedProvider) emitCache(meta obs.Meta, result string) {
	if p.Bus != nil {
		p.Bus.Emit(obs.CacheEvent{Meta: meta, Result: result})
	}
}

func (p *CachedProvider) emitFilters(meta obs.Meta, fo FilterOutcome) {
	if p.Bus == nil {
		return
	}
	for reason, n := range fo.Removed {
		if n > 0 {
			p.Bus.Emit(obs.FilterEvent{Meta: meta, Reason: reason, Count: n})
		}
	}
}
