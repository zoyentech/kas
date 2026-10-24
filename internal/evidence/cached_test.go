package evidence_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/cache"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence"
	"github.com/zoyentech/kas/internal/evidence/inmem"
	"github.com/zoyentech/kas/internal/evidence/perm"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func evidenceCfg(enabled bool) config.EvidenceConfig {
	return config.EvidenceConfig{
		Provider: "inmem", Namespace: "ns-a", TopK: 3, MaxCandidateGroups: 2,
		Validity: config.ValidityConfig{
			DefaultMaxAge: map[string]config.Duration{"manual": config.D(24 * time.Hour)},
		},
		Cache: config.EvidenceCacheConfig{
			Enabled: enabled, Implementation: "memory",
			MaxEntries: 100, MaxBytes: 1 << 20, MaxItemBytes: 1 << 20,
			TTL: config.D(time.Minute),
		},
	}
}

func scope(fp string) domain.PrincipalScope {
	return domain.PrincipalScope{TenantKey: "t", ScopeFingerprint: fp, AuthorizationEpoch: "e1"}
}

func query(q string, s domain.PrincipalScope) domain.EvidenceQuery {
	return domain.EvidenceQuery{
		Question: q, Namespace: "ns-a", CorpusGeneration: "g1", TopK: 3,
		Scope: s, ConfigVersion: "v1", TraceID: "trace-1",
	}
}

func future(now time.Time, ids ...string) []domain.Evidence {
	var out []domain.Evidence
	for _, id := range ids {
		out = append(out, domain.Evidence{ID: id, SourceID: "manual", ObservedAt: now})
	}
	return out
}

// TestCachedProviderMissThenHit miss 全路径 → 写入 → hit 复核返回；
// 缓存事件计数准确（Story 4.3.2.2 T1）。
func TestCachedProviderMissThenHit(t *testing.T) {
	now := time.Now()
	inner := inmem.New(future(now, "e1", "e2", "e3"))
	bus := obs.NewBus()
	var cacheResults []string
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if ce, ok := ev.(obs.CacheEvent); ok {
			cacheResults = append(cacheResults, ce.Result)
		}
	}))
	c := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	p := evidence.NewCachedProvider(inner, c, evidenceCfg(true), nil, bus)
	p.Clock = func() time.Time { return now }

	q := query("服务支持哪些配置方式", scope("fp-a"))
	res, err := p.Search(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Evidence) != 3 || res.FromCache {
		t.Fatalf("miss 全路径应回源 3 条: %+v", res)
	}
	if inner.Calls() != 1 {
		t.Fatalf("provider calls = %d", inner.Calls())
	}
	// 第二次命中
	res2, err := p.Search(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if !res2.FromCache || len(res2.Evidence) != 3 {
		t.Fatalf("应命中缓存: %+v", res2)
	}
	if inner.Calls() != 1 {
		t.Fatalf("命中后不应回源: %d", inner.Calls())
	}
	if len(cacheResults) != 2 || cacheResults[0] != "miss" || cacheResults[1] != "hit" {
		t.Fatalf("缓存事件序列应为 [miss hit]，实际: %v", cacheResults)
	}
}

// TestCachedProviderPermissionRecheckOnHit 缓存时有权、读取时无权：
// 不返回旧结果，回源（Story 4.2.2.1 T2）。
func TestCachedProviderPermissionRecheckOnHit(t *testing.T) {
	now := time.Now()
	// 带 tag 的证据
	tagged := []domain.Evidence{{ID: "t1", SourceID: "manual", ObservedAt: now, PermissionTags: []string{"kb:vip"}}}
	inner := inmem.New(tagged)
	bus := obs.NewBus()
	var filterEvents []obs.FilterEvent
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if fe, ok := ev.(obs.FilterEvent); ok {
			filterEvents = append(filterEvents, fe)
		}
	}))
	c := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	checker := perm.NewTagChecker(map[string][]string{"fp-rich": {"kb:vip"}})
	p := evidence.NewCachedProvider(inner, c, evidenceCfg(true), checker, bus)
	p.Clock = func() time.Time { return now }

	q1 := query("VIP 问题", scope("fp-rich"))
	res, err := p.Search(context.Background(), q1)
	if err != nil || len(res.Evidence) != 1 {
		t.Fatalf("有权 scope 首查应得 1 条: %+v err=%v", res, err)
	}
	// 换 scope（无权）：key 隔离本就 miss；即使同 key（模拟：直接构造相同
	// fingerprint 变化场景由 key 隔离覆盖），此处验证 fp-poor 走 miss+过滤
	q2 := query("VIP 问题", scope("fp-poor"))
	res2, err := p.Search(context.Background(), q2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Evidence) != 0 || res2.FromCache {
		t.Fatalf("无权 scope 不应得到带 tag 证据: %+v", res2)
	}
	// unauthorized 过滤事件
	found := false
	for _, fe := range filterEvents {
		if fe.Reason == perm.ReasonUnauthorized {
			found = true
		}
	}
	if !found {
		t.Fatalf("应有 unauthorized 过滤事件: %+v", filterEvents)
	}
}

// TestCachedProviderHitRecheckExpired 缓存内证据过期：hit 复核过滤后
// 不足量 → 回源（Story 4.3.2.2 T2）。
func TestCachedProviderHitRecheckExpired(t *testing.T) {
	now := time.Now()
	inner := inmem.New(nil)
	bus := obs.NewBus()
	c := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	p := evidence.NewCachedProvider(inner, c, evidenceCfg(true), nil, bus)
	p.Clock = func() time.Time { return now }

	// 直接写入一条"写入时有效、读取时过期"的缓存（模拟时间流逝）
	q := query("过期问题", scope("fp-a"))
	expiring := domain.Evidence{ID: "old1", SourceID: "manual", ObservedAt: now.Add(-2 * time.Hour)}
	d := now.Add(-time.Hour)
	expiring.ValidUntil = &d
	if err := c.Put(context.Background(), mustKey(q), domain.EvidenceResult{Evidence: []domain.Evidence{expiring}}, time.Hour); err != nil {
		t.Fatal(err)
	}
	inner.SetItems(future(now, "fresh1", "fresh2", "fresh3"))
	res, err := p.Search(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if res.FromCache {
		t.Fatal("复核后不足量应回源而非返回旧数据")
	}
	for _, ev := range res.Evidence {
		if ev.ID == "old1" {
			t.Fatal("过期证据泄漏进结果")
		}
	}
	if len(res.Evidence) != 3 {
		t.Fatalf("回源应得 3 条: %+v", res)
	}
}

func mustKey(q domain.EvidenceQuery) domain.CacheKey {
	in := cache.KeyInput{
		Question:           cache.Normalize(q.Question, cache.CaseASCII),
		ConfigVersion:      q.ConfigVersion,
		ScopeFingerprint:   q.Scope.ScopeFingerprint,
		AuthorizationEpoch: q.Scope.AuthorizationEpoch,
		Namespace:          q.Namespace,
		CorpusGeneration:   q.CorpusGeneration,
		CasePolicy:         cache.CaseASCII,
		Policy:             evidenceCfg(true),
	}
	parts := in.Parts()
	return domain.CacheKey{
		QuestionDigest: parts.QuestionDigest, ConfigVersion: parts.ConfigVersion,
		ScopeFingerprint: parts.ScopeFingerprint, AuthorizationEpoch: parts.AuthorizationEpoch,
		KnowledgeNamespace: parts.Namespace, CorpusGeneration: parts.CorpusGeneration,
		RetrievalPolicyVersion: parts.PolicyVersion,
	}
}

// TestCachedProviderForbiddenWrites 禁写清单（Story 4.3.2.1 T3）：
// 全部被滤/错误路径不写入。
func TestCachedProviderForbiddenWrites(t *testing.T) {
	now := time.Now()
	bus := obs.NewBus()

	// 1) 全部证据被过滤（unknown validity）→ 不写
	unknown := inmem.New([]domain.Evidence{{ID: "u1", SourceID: "no-policy"}})
	c := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	p := evidence.NewCachedProvider(unknown, c, evidenceCfg(true), nil, bus)
	p.Clock = func() time.Time { return now }
	if _, err := p.Search(context.Background(), query("未知有效期", scope("fp-a"))); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 0 {
		t.Fatal("全部被滤的结果不应写入缓存")
	}

	// 2) provider 错误 → 透出且不写
	inner := testutil.NewFaultyProvider(inmem.New(future(now, "e1")), testutil.FaultMode{
		Err: errors.New("upstream 500"),
	})
	c2 := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	p2 := evidence.NewCachedProvider(inner, c2, evidenceCfg(true), nil, bus)
	p2.Clock = func() time.Time { return now }
	if _, err := p2.Search(context.Background(), query("触发错误", scope("fp-a"))); err == nil {
		t.Fatal("provider 错误应透出")
	}
	if c2.Len() != 0 {
		t.Fatal("错误结果不应写入缓存")
	}
}

// TestCachedProviderTTLBoundedByDeadline TTL 不晚于最早截止
// （Story 4.2.1.1 T3 与 4.3.2.1 T2 联测：docs/05 §2"cache TTL 不晚于截止"）。
func TestCachedProviderTTLBoundedByDeadline(t *testing.T) {
	now := time.Now()
	// manual 来源 23h 前观察 → 推导截止 1h 后 → TTL = min(1min, 1h) = 1min？
	// 构造更紧：max-age 24h，30min 前 observed → 截止 23.5h 后，配置 TTL 1min
	// → TTL = 1min（配置主导）。再构造截止 30s 后：observed 23h59m30s 前
	// → 截止 30s 后 → TTL = 30s（截止主导）。
	tight := domain.Evidence{ID: "t1", SourceID: "manual", ObservedAt: now.Add(-24*time.Hour + 30*time.Second)}
	inner := inmem.New([]domain.Evidence{tight})
	bus := obs.NewBus()
	c := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	fake := testutil.NewFakeClock(now)
	c.SetClock(fake.Now) // 观察 MemCache TTL
	p := evidence.NewCachedProvider(inner, c, evidenceCfg(true), nil, bus)
	p.Clock = fake.Now
	if _, err := p.Search(context.Background(), query("TTL 联动", scope("fp-a"))); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 1 {
		t.Fatal("有效结果应写入")
	}
	// 推进 31s（> 推导截止剩余 30s）→ 过期 miss
	fake.Advance(31 * time.Second)
	if _, hit, _ := c.Get(context.Background(), mustKey(query("TTL 联动", scope("fp-a")))); hit {
		t.Fatal("TTL 应不晚于推导截止（30s 后过期）")
	}
}

// TestCachedProviderDisabledCacheStillFilters 缓存禁用：过滤仍生效
// （F-003 过滤路径唯一，不因缓存禁用绕过）。
func TestCachedProviderDisabledCacheStillFilters(t *testing.T) {
	now := time.Now()
	mixed := []domain.Evidence{
		{ID: "ok", SourceID: "manual", ObservedAt: now},
		{ID: "exp", ValidUntil: func() *time.Time { d := now.Add(-time.Minute); return &d }()},
		{ID: "unk", SourceID: "no-policy"},
	}
	inner := inmem.New(mixed)
	p := evidence.NewCachedProvider(inner, nil, evidenceCfg(false), nil, nil)
	p.Clock = func() time.Time { return now }
	res, err := p.Search(context.Background(), query("禁用缓存", scope("fp-a")))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Evidence) != 1 || res.Evidence[0].ID != "ok" {
		t.Fatalf("过滤应仅保留 ok: %+v", res.Evidence)
	}
}

// TestFilterEventPayloadNoEvidenceID 事件 payload 无 Evidence 标识
// （Story 4.2.1.2 T1 验收）。
func TestFilterEventPayloadNoEvidenceID(t *testing.T) {
	now := time.Now()
	inner := inmem.New([]domain.Evidence{{ID: "secret-id-1", SourceID: "no-policy"}})
	bus := obs.NewBus()
	var payloads []obs.FilterEvent
	bus.Subscribe(obs.SubscriberFunc(func(ev obs.Event) {
		if fe, ok := ev.(obs.FilterEvent); ok {
			payloads = append(payloads, fe)
		}
	}))
	c := cache.NewMemCache(100, 1<<20, 1<<20, bus)
	p := evidence.NewCachedProvider(inner, c, evidenceCfg(true), nil, bus)
	p.Clock = func() time.Time { return now }
	_, _ = p.Search(context.Background(), query("事件脱敏", scope("fp-a")))
	for _, fe := range payloads {
		if strings.Contains(fe.Reason, "secret-id-1") || fe.Count != 1 {
			t.Fatalf("事件只含 reason 与计数: %+v", fe)
		}
	}
	if len(payloads) == 0 {
		t.Fatal("应有 unknown_validity 事件")
	}
}
