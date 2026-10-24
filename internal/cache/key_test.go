package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func baseInput() KeyInput {
	return KeyInput{
		Question:           "服务支持哪些配置方式",
		ConfigVersion:      "v1",
		ScopeFingerprint:   "scope-fp-a",
		AuthorizationEpoch: "epoch-1",
		Namespace:          "ns-a",
		CorpusGeneration:   "gen-1",
		CasePolicy:         CaseASCII,
		Policy:             config.EvidenceConfig{TopK: 8, MaxCandidateGroups: 4},
	}
}

// TestKeyDimensionIsolation 七维逐一变化 → 字节串与摘要都变化
// （Story 4.3.1.2 T1；docs/05 §2 隔离矩阵）。
func TestKeyDimensionIsolation(t *testing.T) {
	base := baseInput()
	baseCombined := string(Combine(base.Parts()))
	baseDigest := Digest(base.Parts())

	mutations := map[string]func(*KeyInput){
		"question":          func(in *KeyInput) { in.Question = "怎么配置超时" },
		"config_version":    func(in *KeyInput) { in.ConfigVersion = "v2" },
		"scope":             func(in *KeyInput) { in.ScopeFingerprint = "scope-fp-b" },
		"epoch":             func(in *KeyInput) { in.AuthorizationEpoch = "epoch-2" },
		"namespace":         func(in *KeyInput) { in.Namespace = "ns-b" },
		"corpus_generation": func(in *KeyInput) { in.CorpusGeneration = "gen-2" },
		"policy":            func(in *KeyInput) { in.Policy.TopK = 9 },
	}
	for name, mutate := range mutations {
		in := baseInput()
		mutate(&in)
		if got := string(Combine(in.Parts())); got == baseCombined {
			t.Errorf("%s 变化未改变组合字节串", name)
		}
		if got := Digest(in.Parts()); got == baseDigest {
			t.Errorf("%s 变化未改变存储摘要", name)
		}
	}
}

// TestKeyPolicyVersionIncludesValidity 策略版本包含 max-age 表（策略变化换 key）。
func TestKeyPolicyVersionIncludesValidity(t *testing.T) {
	in1 := baseInput()
	in2 := baseInput()
	in2.Policy.Validity.DefaultMaxAge = map[string]config.Duration{
		"product_manual": config.D(time.Hour),
	}
	if PolicyVersion(in1.Policy) == PolicyVersion(in2.Policy) {
		t.Fatal("max-age 策略变化应改变策略版本")
	}
}

// TestKeyDigestNoPlaintext 存储形态无明文：Combined/Digest 输出不含问题原文
// （Story 4.3.1.2 T2 扫描测试）。
func TestKeyDigestNoPlaintext(t *testing.T) {
	in := baseInput()
	combined := string(Combine(in.Parts()))
	digest := Digest(in.Parts())
	for _, out := range []string{combined, digest} {
		if contains(out, in.Question) || contains(out, "服务支持") {
			t.Fatal("key 形态泄漏问题原文")
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// memKey 把 KeyInput 转为 domain.CacheKey（与 CachedProvider 同口径）。
func memKey(in KeyInput) domain.CacheKey {
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

// TestCacheKeyIsolationMatrix F-004：单维变化全 miss（docs/05 §2 隔离行）。
func TestCacheKeyIsolationMatrix(t *testing.T) {
	c := NewMemCache(100, 1<<20, 1<<20, nil)
	base := baseInput()
	if err := c.Put(context.Background(), memKey(base), resultOf("e1"), time.Minute); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*KeyInput){
		"question":          func(in *KeyInput) { in.Question = "另一个问题" },
		"config_version":    func(in *KeyInput) { in.ConfigVersion = "v2" },
		"scope":             func(in *KeyInput) { in.ScopeFingerprint = "scope-b" },
		"epoch":             func(in *KeyInput) { in.AuthorizationEpoch = "e2" },
		"namespace":         func(in *KeyInput) { in.Namespace = "ns-b" },
		"corpus_generation": func(in *KeyInput) { in.CorpusGeneration = "g2" },
	}
	for name, mutate := range mutations {
		in := baseInput()
		mutate(&in)
		if _, hit, _ := c.Get(context.Background(), memKey(in)); hit {
			t.Errorf("F-004 隔离失败：%s 变化后仍命中", name)
		}
	}
	// 原始 key 命中
	if _, hit, _ := c.Get(context.Background(), memKey(base)); !hit {
		t.Fatal("原始 key 应命中")
	}
}

// TestDeleteNamespace namespace 主动清理生效（Story 4.3.1.2 T3）。
func TestDeleteNamespace(t *testing.T) {
	c := NewMemCache(100, 1<<20, 1<<20, nil)
	for _, ns := range []string{"ns-a", "ns-b"} {
		in := baseInput()
		in.Namespace = ns
		if err := c.Put(context.Background(), memKey(in), resultOf("e-"+ns), time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.DeleteNamespace(context.Background(), "ns-a"); err != nil {
		t.Fatal(err)
	}
	in := baseInput()
	if _, hit, _ := c.Get(context.Background(), memKey(in)); hit {
		t.Fatal("ns-a 清理后不应命中")
	}
	in.Namespace = "ns-b"
	if _, hit, _ := c.Get(context.Background(), memKey(in)); !hit {
		t.Fatal("ns-b 不应被误删")
	}
}

func resultOf(ids ...string) domain.EvidenceResult {
	var ev []domain.Evidence
	for _, id := range ids {
		ev = append(ev, domain.Evidence{ID: id, SourceID: "s1"})
	}
	return domain.EvidenceResult{Evidence: ev}
}

// TestMemCacheLRUDeterministic 确定性 LRU：相同操作序列淘汰结果确定
// （Story 4.3.2.1 T1）。
func TestMemCacheLRUDeterministic(t *testing.T) {
	c := NewMemCache(2, 1<<20, 1<<20, nil)
	k1, k2, k3 := memKeyWithQuestion("q1"), memKeyWithQuestion("q2"), memKeyWithQuestion("q3")
	ctx := context.Background()
	_ = c.Put(ctx, k1, resultOf("e1"), time.Minute)
	_ = c.Put(ctx, k2, resultOf("e2"), time.Minute)
	// 触碰 k1（提升）后插入 k3 → 淘汰 k2
	if _, hit, _ := c.Get(ctx, k1); !hit {
		t.Fatal("k1 应命中")
	}
	_ = c.Put(ctx, k3, resultOf("e3"), time.Minute)
	if _, hit, _ := c.Get(ctx, k2); hit {
		t.Fatal("k2 应被 LRU 淘汰")
	}
	if _, hit, _ := c.Get(ctx, k1); !hit {
		t.Fatal("k1 应保留")
	}
	if c.Len() != 2 {
		t.Fatalf("条目数 = %d, want 2", c.Len())
	}
}

// TestMemCacheTripleLimits 三重上限各自用例（含恰好等于上限）。
func TestMemCacheTripleLimits(t *testing.T) {
	ctx := context.Background()
	// 单项字节上限：超限拒存、恰好等于通过
	small := NewMemCache(10, 1<<20, int64(serializeLen(resultOf("e1"))), nil)
	if err := small.Put(ctx, memKeyWithQuestion("q1"), resultOf("e1", "e2"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := small.Get(ctx, memKeyWithQuestion("q1")); hit {
		t.Fatal("超单项上限应拒存")
	}
	if err := small.Put(ctx, memKeyWithQuestion("q2"), resultOf("e1"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := small.Get(ctx, memKeyWithQuestion("q2")); !hit {
		t.Fatal("恰好等于单项上限应可存")
	}
	// entries 上限
	lim := NewMemCache(2, 1<<20, 1<<20, nil)
	for i := 0; i < 5; i++ {
		_ = lim.Put(ctx, memKeyWithQuestion(string(rune('a'+i))), resultOf("e"), time.Minute)
	}
	if lim.Len() != 2 {
		t.Fatalf("entries 上限 = %d, want 2", lim.Len())
	}
	// bytes 上限：单值 1MB 上限下两条大值只留一条
	big := func() domain.EvidenceResult { return resultOf("big-big-big") }
	byteLim := NewMemCache(10, int64(serializeLen(big()))+1, 1<<20, nil)
	_ = byteLim.Put(ctx, memKeyWithQuestion("q1"), big(), time.Minute)
	_ = byteLim.Put(ctx, memKeyWithQuestion("q2"), big(), time.Minute)
	if byteLim.Len() != 1 {
		t.Fatalf("bytes 上限后条目 = %d, want 1", byteLim.Len())
	}
}

// TestMemCacheTTLExpiry TTL 过期读取视为 miss 并清理。
func TestMemCacheTTLExpiry(t *testing.T) {
	c := NewMemCache(10, 1<<20, 1<<20, nil)
	ctx := context.Background()
	k := memKeyWithQuestion("q1")
	// 用可注入时钟
	fake := testutil.NewFakeClock(time.Now())
	c.clock = fake.Now
	_ = c.Put(ctx, k, resultOf("e1"), 100*time.Millisecond)
	if _, hit, _ := c.Get(ctx, k); !hit {
		t.Fatal("TTL 内应命中")
	}
	fake.Advance(150 * time.Millisecond)
	if _, hit, _ := c.Get(ctx, k); hit {
		t.Fatal("TTL 过期应 miss")
	}
	if c.Len() != 0 {
		t.Fatal("过期条目应被清理")
	}
}

// TestMemCacheTTLPutNonPositive TTL ≤ 0 不写（bypass）。
func TestMemCacheTTLPutNonPositive(t *testing.T) {
	c := NewMemCache(10, 1<<20, 1<<20, nil)
	_ = c.Put(context.Background(), memKeyWithQuestion("q1"), resultOf("e1"), 0)
	_ = c.Put(context.Background(), memKeyWithQuestion("q2"), resultOf("e2"), -time.Second)
	if c.Len() != 0 {
		t.Fatal("ttl≤0 不应写入")
	}
	if c.Stats().Bypasses != 2 {
		t.Fatalf("bypass 计数 = %d, want 2", c.Stats().Bypasses)
	}
}

// TestTTLDecisionMin 三来源各自主导。
func TestTTLDecisionMin(t *testing.T) {
	now := time.Now()
	// 配置 TTL 主导（最小）
	if got := TTLDecision(30*time.Second, time.Hour, now.Add(time.Hour), now); got != 30*time.Second {
		t.Fatalf("configured 主导: %s", got)
	}
	// 来源新鲜度主导
	if got := TTLDecision(30*time.Second, 10*time.Second, now.Add(time.Hour), now); got != 10*time.Second {
		t.Fatalf("freshness 主导: %s", got)
	}
	// 最早截止主导
	if got := TTLDecision(30*time.Second, time.Hour, now.Add(5*time.Second), now); got != 5*time.Second {
		t.Fatalf("deadline 主导: %s", got)
	}
	// ≤0 → 0（不写入）
	if got := TTLDecision(30*time.Second, time.Hour, now.Add(-time.Second), now); got != 0 {
		t.Fatalf("负截止应返回 0: %s", got)
	}
}

// TestMemCacheConcurrentReadWrite 并发读写 -race + 计数一致
// （Story 4.3.2.2 T3）。
func TestMemCacheConcurrentReadWrite(t *testing.T) {
	c := NewMemCache(50, 1<<20, 1<<20, nil)
	ctx := context.Background()
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				in := baseInput()
				in.Question = "q" + string(rune('a'+(i%10)))
				_, _, _ = c.Get(ctx, memKey(in))
				_ = c.Put(ctx, memKey(in), resultOf("e"), time.Minute)
			}
		}(w)
	}
	wg.Wait()
	if c.Len() > 50 {
		t.Fatalf("条目超上限: %d", c.Len())
	}
}

func memKeyWithQuestion(q string) domain.CacheKey {
	in := baseInput()
	in.Question = q
	return memKey(in)
}

// TestDigestOf 与 Digest 同口径；EarliestDeadline 空集与多值。
func TestDigestOfAndEarliestDeadline(t *testing.T) {
	in := baseInput()
	if in.DigestOf() != Digest(in.Parts()) {
		t.Fatal("DigestOf 与 Digest(Parts()) 必须同口径")
	}
	// EarliestDeadline：空集零值；多值取最早
	if !EarliestDeadline(nil).IsZero() {
		t.Fatal("空集应返回零值")
	}
	t1 := time.Now()
	t2 := t1.Add(time.Hour)
	if got := EarliestDeadline(map[string]time.Time{"a": t2, "b": t1}); !got.Equal(t1) {
		t.Fatalf("应取最早: %v", got)
	}
}
