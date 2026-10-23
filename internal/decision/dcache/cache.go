// Package dcache 实现受约束决策缓存（Story 5.3.2.1；docs/design/04 §3）：
// key 绑定 config/model/policy 版本、候选 fingerprint、知识代次与
// principal scope；命中后再次校验 action 与目标组；与 Evidence 缓存互不
// 影响。只缓存可复用的正向决策；abstain/clarify 终态不入缓存。
package dcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kas-runtime/kas/internal/decision"
	"github.com/kas-runtime/kas/internal/domain"
)

// Key 是决策缓存 key 维度（六维，Story 5.3.2.1 T1）。
type Key struct {
	ConfigVersion string
	Model         string
	ModelVersion  string
	CandidatesFP  string // 候选组 fingerprint
	CorpusGen     string
	ScopeFP       string // principal scope fingerprint
}

// CandidatesFingerprint 计算候选组 fingerprint：组 ID 集 + 证据版本摘要
// 的规范化哈希（顺序敏感——候选排序是决策语义的一部分）。
func CandidatesFingerprint(groups []domain.CandidateGroup) string {
	type flat struct {
		ID       string   `json:"id"`
		EvIDs    []string `json:"ev"`
		Versions []string `json:"ver"`
		Score    float64  `json:"score"`
	}
	flats := make([]flat, 0, len(groups))
	for _, g := range groups {
		f := flat{ID: g.ID, EvIDs: make([]string, 0, len(g.Evidence)), Versions: make([]string, 0, len(g.Evidence)), Score: g.Score}
		for _, e := range g.Evidence {
			f.EvIDs = append(f.EvIDs, e.EvidenceID)
			f.Versions = append(f.Versions, e.ContentVersion)
		}
		flats = append(flats, f)
	}
	b, err := json.Marshal(flats)
	if err != nil {
		return "fingerprint-error"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:24]
}

func (k Key) digest() string {
	b, _ := json.Marshal(k)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Cache 是 LRU+TTL 决策缓存（并发安全）。
type Cache struct {
	mu         sync.Mutex
	entries    map[string]*cacheEntry
	order      []string // LRU 顺序（尾部最旧）
	maxEntries int
	ttl        time.Duration
	clock      func() time.Time
}

type cacheEntry struct {
	value     domain.Decision
	expiresAt time.Time
}

// New 构建决策缓存。
func New(maxEntries int, ttl time.Duration) *Cache {
	return &Cache{entries: map[string]*cacheEntry{}, maxEntries: maxEntries, ttl: ttl, clock: time.Now}
}

// SetClock 注入时钟（测试）。
func (c *Cache) SetClock(fn func() time.Time) { c.clock = fn }

// Provider 包装为 DecisionProvider（链第三级；Provider 名 decision_cache）。
type Provider struct {
	cache *Cache
}

// NewProvider 构建缓存级 provider。
func NewProvider(maxEntries int, ttl time.Duration) *Provider {
	return &Provider{cache: New(maxEntries, ttl)}
}

// ErrMiss 是未命中哨兵（链继续下一级）。
var ErrMiss = fmt.Errorf("kas/dcache: miss")

// Get 查询（带命中再校验，Story 5.3.2.1 T2）：用当前输入重过
// ValidateDecision；不通过 → 淘汰条目返回 miss。
func (p *Provider) Get(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	c := p.cache
	k := KeyFor(in)
	c.mu.Lock()
	e, ok := c.entries[k.digest()]
	if ok && !c.clock().Before(e.expiresAt) {
		c.removeLocked(k.digest())
		ok = false
	}
	var d domain.Decision
	if ok {
		d = e.value
		c.touchLocked(k.digest())
	}
	c.mu.Unlock()
	if !ok {
		return domain.Decision{}, ErrMiss
	}
	if err := decision.ValidateDecision(in, d); err != nil {
		// 命中但校验失败：淘汰（伪造越界目标的条目被拒）
		p.cache.mu.Lock()
		p.cache.removeLocked(k.digest())
		p.cache.mu.Unlock()
		return domain.Decision{}, ErrMiss
	}
	d.Provider = "decision_cache"
	d.Degraded = true
	return d, nil
}

// Put 写入（禁写规则：abstain/clarify 终态与低置信结果不入缓存，
// Story 5.3.2.1 T3——只缓存可复用的正向决策）。
func (p *Provider) Put(ctx context.Context, in domain.DecisionInput, d domain.Decision) {
	if d.Action == domain.ActionAbstain || d.Action == domain.ActionClarify {
		return
	}
	const minReusableConfidence = 0.5
	if d.Confidence < minReusableConfidence {
		return
	}
	c := p.cache
	k := KeyFor(in).digest()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.order) >= c.maxEntries {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
	if _, exists := c.entries[k]; !exists {
		c.order = append(c.order, k)
	}
	c.entries[k] = &cacheEntry{value: d, expiresAt: c.clock().Add(c.ttl)}
}

// Decide 实现 DecisionProvider：Get 命中返回；miss 透出 ErrMiss 由链继续。
func (p *Provider) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	return p.Get(ctx, in)
}

// Health 恒 healthy（内存缓存无外部依赖）。
func (p *Provider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// KeyFor 由决策输入组装 key（model/model_version 取 ModelHint 解析或
// 由链写入 Decision 后回填；决策前使用 hint 维度）。
func KeyFor(in domain.DecisionInput) Key {
	return Key{
		ConfigVersion: in.ConfigVersion,
		CandidatesFP:  CandidatesFingerprint(in.Groups),
		CorpusGen:     in.KnowledgeCorpusGen,
		ScopeFP:       in.ScopeFingerprint,
	}
}

// KeyForFull 是含 model 维度的完整 key（链在拿到上游决策后回填缓存用）。
func KeyForFull(in domain.DecisionInput, model, modelVersion string) Key {
	k := KeyFor(in)
	k.Model, k.ModelVersion = model, modelVersion
	return k
}

func (c *Cache) removeLocked(digest string) {
	delete(c.entries, digest)
	for i, k := range c.order {
		if k == digest {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *Cache) touchLocked(digest string) {
	for i, k := range c.order {
		if k == digest {
			c.order = append(append(c.order[:i], c.order[i+1:]...), digest)
			break
		}
	}
}

// Len 返回当前条目数（测试与容量断言）。
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// GetUnderlying 暴露内部缓存（链回填与测试用）。
func (p *Provider) GetUnderlying() *Cache { return p.cache }
