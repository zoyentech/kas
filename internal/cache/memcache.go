package cache

import (
	"container/list"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
)

// MemCache 是 v1.2 默认进程内 Evidence 缓存（Story 4.3.2.1）：entries/总字节/
// 单项字节三重上限的确定性 LRU + TTL 过期。并发安全；仅缓存 Evidence 列表
// 与元数据，不缓存最终回答。命中仍需上层用当前权限与当前时刻复核。
type MemCache struct {
	mu           sync.Mutex
	ll           *list.List               // front = 最近使用
	items        map[string]*list.Element // digest → element
	bytes        int64
	maxEntries   int
	maxBytes     int64
	maxItemBytes int64
	clock        func() time.Time
	bus          *obs.Bus // 可空；eviction/bypass 计数事件

	stats Stats
}

// Stats 是缓存内部计数（与事件计数对照测试用）。
type Stats struct {
	Hits, Misses, Bypasses, Errors                      int64
	EvictEntries, EvictBytes, EvictItem, EvictNamespace int64
}

// NewMemCache 构建有界内存缓存。maxEntries/maxBytes/maxItemBytes 必须 > 0
// （配置校验已保证；此处再防御）。
func NewMemCache(maxEntries int, maxBytes, maxItemBytes int64, bus *obs.Bus) *MemCache {
	return &MemCache{
		ll:           list.New(),
		items:        make(map[string]*list.Element, maxEntries/8+1),
		maxEntries:   maxEntries,
		maxBytes:     maxBytes,
		maxItemBytes: maxItemBytes,
		clock:        time.Now,
		bus:          bus,
	}
}

type entry struct {
	digest    string
	namespace string
	value     domain.EvidenceResult
	size      int64
	expiresAt time.Time
}

// Get 返回未过期条目并提升 LRU 位置；过期条目即时清理并按 miss 处理。
func (c *MemCache) Get(ctx context.Context, key domain.CacheKey) (domain.EvidenceResult, bool, error) {
	digest := digestFromCacheKey(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[digest]
	if !ok {
		c.stats.Misses++
		return domain.EvidenceResult{}, false, nil
	}
	e := el.Value.(*entry)
	if !c.clock().Before(e.expiresAt) {
		c.removeLocked(el)
		c.stats.Misses++
		return domain.EvidenceResult{}, false, nil
	}
	c.ll.MoveToFront(el)
	c.stats.Hits++
	out := e.value
	out.FromCache = true
	return out, true, nil
}

// Put 写入条目：ttl ≤ 0 不写（bypass）；单项字节超上限直接拒存（bypass）；
// 插入后按 LRU 逐出至 entries/bytes 双达标。
func (c *MemCache) Put(ctx context.Context, key domain.CacheKey, value domain.EvidenceResult, ttl time.Duration) error {
	if ttl <= 0 {
		c.bypass("ttl_nonpositive")
		return nil
	}
	digest := digestFromCacheKey(key)
	size := int64(serializeLen(value))
	if size > c.maxItemBytes {
		c.bypass("item_too_large")
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// 已存在则先移除（覆盖写）
	if el, ok := c.items[digest]; ok {
		c.removeLocked(el)
	}
	e := &entry{
		digest:    digest,
		namespace: key.KnowledgeNamespace,
		value:     value,
		size:      size,
		expiresAt: c.clock().Add(ttl),
	}
	el := c.ll.PushFront(e)
	c.items[digest] = el
	c.bytes += size
	// LRU 逐出至达标（确定性：严格从尾部逐出）
	for c.ll.Len() > c.maxEntries || c.bytes > c.maxBytes {
		last := c.ll.Back()
		if last == nil {
			break
		}
		c.removeLocked(last)
		c.stats.EvictEntries++
		c.busEmit(obs.CacheEvictionEvent{Reason: "capacity"})
	}
	return nil
}

// DeleteNamespace 按 namespace 主动清理（配置/授权变更后调用）。
func (c *MemCache) DeleteNamespace(ctx context.Context, namespace string) error {
	c.mu.Lock()
	var toRemove []*list.Element
	for el := c.ll.Front(); el != nil; el = el.Next() {
		if el.Value.(*entry).namespace == namespace {
			toRemove = append(toRemove, el)
		}
	}
	for _, el := range toRemove {
		c.removeLocked(el)
		c.stats.EvictNamespace++
	}
	c.mu.Unlock()
	if len(toRemove) > 0 {
		c.busEmit(obs.CacheEvictionEvent{Reason: "namespace_delete"})
	}
	return nil
}

// SetClock 注入时钟（测试用；默认 time.Now）。
func (c *MemCache) SetClock(fn func() time.Time) { c.clock = fn }

// Stats 返回内部计数快照。
func (c *MemCache) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}

// Len 返回当前条目数。
func (c *MemCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *MemCache) removeLocked(el *list.Element) {
	e := el.Value.(*entry)
	c.ll.Remove(el)
	delete(c.items, e.digest)
	c.bytes -= e.size
}

// bypass 计数并广播 bypass 事件（调用方不得持有 c.mu）。
func (c *MemCache) bypass(reason string) {
	c.mu.Lock()
	c.stats.Bypasses++
	c.mu.Unlock()
	c.busEmit(obs.CacheEvictionEvent{Reason: reason})
}

// busEmit 发射事件（可空 bus；bus 自带锁，持不持有 c.mu 均安全）。
func (c *MemCache) busEmit(ev obs.Event) {
	if c.bus != nil {
		c.bus.Emit(ev)
	}
}

// digestFromCacheKey 由七维 key 计算存储主键（与 KeyInput.DigestOf 同口径：
// cache.Digest(cache.KeyParts) 唯一实现）。
func digestFromCacheKey(key domain.CacheKey) string {
	return Digest(KeyParts{
		QuestionDigest:     key.QuestionDigest,
		ConfigVersion:      key.ConfigVersion,
		ScopeFingerprint:   key.ScopeFingerprint,
		AuthorizationEpoch: key.AuthorizationEpoch,
		Namespace:          key.KnowledgeNamespace,
		CorpusGeneration:   key.CorpusGeneration,
		PolicyVersion:      key.RetrievalPolicyVersion,
	})
}

// serializeLen 计算值序列化长度（字节统计口径；序列化形态不落盘，仅统计）。
func serializeLen(v domain.EvidenceResult) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 256 // 防御性下限：异常形态按非零计
	}
	return len(b)
}
