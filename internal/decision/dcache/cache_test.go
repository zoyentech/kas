package dcache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kas-runtime/kas/internal/decision/dcache"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func in(scope, corpus, cv string, groups ...domain.CandidateGroup) domain.DecisionInput {
	if groups == nil {
		groups = []domain.CandidateGroup{{ID: "g1", Evidence: []domain.EvidenceRef{{EvidenceID: "e1", ContentVersion: "v1"}}}}
	}
	return domain.DecisionInput{ScopeFingerprint: scope, KnowledgeCorpusGen: corpus, ConfigVersion: cv, Groups: groups}
}

var answer = domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"g1"}, Confidence: 0.9}

// TestDecisionCacheKeyIsolation 六维隔离（Story 5.3.2.1 T1）。
func TestDecisionCacheKeyIsolation(t *testing.T) {
	p := dcache.NewProvider(100, time.Minute)
	base := in("s1", "g1", "v1")
	p.Put(context.Background(), base, answer)

	miss := []domain.DecisionInput{
		in("s2", "g1", "v1"), // scope
		in("s1", "g2", "v1"), // corpus
		in("s1", "g1", "v2"), // config
		in("s1", "g1", "v1", domain.CandidateGroup{ID: "g1", Evidence: []domain.EvidenceRef{{EvidenceID: "e2"}}}), // 候选
	}
	for i, m := range miss {
		if _, err := p.Get(context.Background(), m); !errors.Is(err, dcache.ErrMiss) {
			t.Errorf("维度 %d 变化后不应命中: %v", i, err)
		}
	}
	if d, err := p.Get(context.Background(), base); err != nil || d.Action != domain.ActionAnswer {
		t.Fatalf("原始 key 应命中: %v %v", d, err)
	}
}

// TestDecisionCacheRevalidation 命中再校验：伪造越界目标组的条目被拒
// （Story 5.3.2.1 T2；docs/05 §2 命中行）。
func TestDecisionCacheRevalidation(t *testing.T) {
	p := dcache.NewProvider(100, time.Minute)
	base := in("s1", "g1", "v1")
	// 组 g1；伪造决策目标组 gX（不属于当前输入）
	p.Put(context.Background(), base, domain.Decision{Action: domain.ActionAnswer, TargetGroupIDs: []string{"gX"}, Confidence: 0.9})
	if _, err := p.Get(context.Background(), base); !errors.Is(err, dcache.ErrMiss) {
		t.Fatalf("越界目标条目应被再校验拒绝: %v", err)
	}
	// 被拒后条目已淘汰
	if p.GetUnderlying().Len() != 0 {
		t.Fatal("校验失败条目应淘汰")
	}
}

// TestDecisionCacheTTLAndForbiddenWrites TTL 过期 + 禁写终态/低置信
// （Story 5.3.2.1 T3）。
func TestDecisionCacheTTLAndForbiddenWrites(t *testing.T) {
	fake := testutil.NewFakeClock(time.Now())
	p := dcache.NewProvider(10, 30*time.Second)
	p.GetUnderlying().SetClock(fake.Now)
	base := in("s1", "g1", "v1")
	p.Put(context.Background(), base, answer)

	// abstain/clarify 终态不入缓存
	p.Put(context.Background(), base, domain.Decision{Action: domain.ActionAbstain})
	p.Put(context.Background(), base, domain.Decision{Action: domain.ActionClarify, Confidence: 0.9})
	// 低置信不入缓存
	p2 := dcache.NewProvider(10, time.Minute)
	p2.Put(context.Background(), in("sx", "g", "v"), domain.Decision{Action: domain.ActionAnswer, Confidence: 0.3})
	if p2.GetUnderlying().Len() != 0 {
		t.Fatal("低置信决策不应入缓存")
	}

	// TTL 过期
	fake.Advance(31 * time.Second)
	if _, err := p.Get(context.Background(), base); !errors.Is(err, dcache.ErrMiss) {
		t.Fatalf("TTL 过期应 miss: %v", err)
	}
}

// TestCandidatesFingerprint 候选 fingerprint 顺序敏感与内容敏感。
func TestCandidatesFingerprint(t *testing.T) {
	g1 := []domain.CandidateGroup{{ID: "a", Evidence: []domain.EvidenceRef{{EvidenceID: "e1"}}}, {ID: "b"}}
	g2 := []domain.CandidateGroup{{ID: "b"}, {ID: "a", Evidence: []domain.EvidenceRef{{EvidenceID: "e1"}}}}
	if dcache.CandidatesFingerprint(g1) == dcache.CandidatesFingerprint(g2) {
		t.Fatal("候选顺序是决策语义，fingerprint 应顺序敏感")
	}
	g3 := []domain.CandidateGroup{{ID: "a", Evidence: []domain.EvidenceRef{{EvidenceID: "e1", ContentVersion: "v9"}}}, {ID: "b"}}
	if dcache.CandidatesFingerprint(g1) == dcache.CandidatesFingerprint(g3) {
		t.Fatal("证据版本变化应改变 fingerprint")
	}
}
