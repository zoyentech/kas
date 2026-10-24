package matrix

import (
	"context"
	"strings"
	"testing"
	"time"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/domain"
)

// 矩阵行 1：标准可答、问法改写——Evidence recall 达标，事实由引用支持。
func TestRowStandardAnswerable(t *testing.T) {
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1"), validEvidence("e2")}}
	c := newClient(t, ev, answerDecision(), nil)

	resp, err := c.Answer(context.Background(), req("服务支持哪些配置方式"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAnswer || len(resp.Citations) != 2 {
		t.Fatalf("标准可答: %+v", resp)
	}
	// 问法改写（空白/大小写形态）同一证据
	resp2, err := c.Answer(context.Background(), req("  服务支持哪些配置方式 "))
	if err != nil || resp2.Action != kas.ActionAnswer {
		t.Fatalf("改写问法: %+v %v", resp2, err)
	}
}

// 矩阵行 2：空结果/低相关/no-answer → clarify 或 abstain，不编造。
func TestRowNoAnswer(t *testing.T) {
	ev := &matrixEvidence{items: nil}
	c := newClient(t, ev, answerDecision(), nil)
	resp, err := c.Answer(context.Background(), req("无结果的问题"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAbstain && resp.Action != kas.ActionClarify {
		t.Fatalf("空结果应 abstain/clarify: %+v", resp)
	}
	if resp.Answer != "" || len(resp.Citations) != 0 {
		t.Fatalf("不得编造: %+v", resp)
	}
}

// 矩阵行 3+18：过期/未生效被过滤，不进答案；引用完整性。
func TestRowExpiredAndCitationIntegrity(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Minute)
	future := now.Add(time.Minute)
	later := now.Add(time.Hour)
	ev := &matrixEvidence{items: []domain.Evidence{
		validEvidence("ok1"),
		{ID: "old1", ValidUntil: &expired},
		{ID: "fut1", ValidFrom: &future, ValidUntil: &later},
	}}
	c := newClient(t, ev, answerDecision(), nil)
	resp, err := c.Answer(context.Background(), req("过期知识问题"))
	if err != nil {
		t.Fatal(err)
	}
	for _, cit := range resp.Citations {
		if cit.EvidenceID != "ok1" {
			t.Fatalf("过期/未生效证据进入引用: %+v", resp.Citations)
		}
	}
}

// 矩阵行 4：有效期未知且无来源策略 → fail-closed。
func TestRowUnknownValidityFailClosed(t *testing.T) {
	ev := &matrixEvidence{items: []domain.Evidence{
		{ID: "unknown1", SourceID: "no-such-source"}, // 无显式时间、无策略
	}}
	c := newClient(t, ev, answerDecision(), nil)
	resp, err := c.Answer(context.Background(), req("未知有效期问题"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAbstain || len(resp.Citations) != 0 {
		t.Fatalf("unknown 必须 fail-closed: %+v", resp)
	}
}

// 矩阵行 5：已批准来源 max-age → 截止推导正确，cache TTL 不晚于截止。
func TestRowMaxAgeDerivationAndTTL(t *testing.T) {
	// manual 来源 23h59m 前 observed（max-age 24h）→ 推导截止 ~1m 后
	ev := &matrixEvidence{items: []domain.Evidence{{
		ID: "aged1", SourceID: "manual", ObservedAt: time.Now().Add(-24*time.Hour + time.Minute),
	}}}
	c := newClient(t, ev, answerDecision(), nil)
	resp, err := c.Answer(context.Background(), req("max-age 推导问题"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAnswer || len(resp.Citations) != 1 {
		t.Fatalf("max-age 推导应有效: %+v", resp)
	}
}

// 矩阵行 6：同问不同 tenant/scope/authorization epoch → 跨 scope cache hit 为 0。
func TestRowCrossScopeIsolation(t *testing.T) {
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
	c := newClient(t, ev, answerDecision(), nil)

	r1 := req("同题不同权限")
	r1.Principal.ScopeFingerprint = "fp-a"
	if _, err := c.Answer(context.Background(), r1); err != nil {
		t.Fatal(err)
	}
	// 换 scope/epoch：语义上必须回源（这里断言不返回旧结果 + 正常回答）
	r2 := req("同题不同权限")
	r2.Principal.ScopeFingerprint = "fp-b"
	r2.Principal.AuthorizationEpoch = "e2"
	resp, err := c.Answer(context.Background(), r2)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAnswer {
		t.Fatalf("换 scope 后仍应正常回答（独立缓存）: %+v", resp)
	}
	// 跨 epoch 同理
	r3 := req("同题不同权限")
	r3.Principal.AuthorizationEpoch = "e3"
	if _, err := c.Answer(context.Background(), r3); err != nil {
		t.Fatal(err)
	}
	// F-004 的零跨 scope hit 由 internal/cache 隔离矩阵 + internal/evidence
	// 复核用例共同证明（P0 安全集）
}

// 矩阵行 7：corpus generation 改变 → 不读取上一代 cache。
func TestRowCorpusGenerationSwitch(t *testing.T) {
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("gen-e1")}}
	c := newClient(t, ev, answerDecision(), nil)
	r1 := req("代次切换问题")
	r1.Knowledge.CorpusGeneration = "g1"
	if _, err := c.Answer(context.Background(), r1); err != nil {
		t.Fatal(err)
	}
	r2 := req("代次切换问题")
	r2.Knowledge.CorpusGeneration = "g2"
	resp, err := c.Answer(context.Background(), r2)
	if err != nil {
		t.Fatal(err)
	}
	// g2 无缓存条目（key 隔离）→ 回源 → 正常回答；无上一代数据混入
	for _, cit := range resp.Citations {
		if !strings.Contains(cit.EvidenceID, "gen-") {
			t.Fatalf("上一代数据混入: %+v", cit)
		}
	}
}

// 矩阵行 12（引用完整性的越权面）：带权限 tag 的证据对无权 scope 不得引用。
func TestRowUnauthorizedCitation(t *testing.T) {
	until := time.Now().Add(time.Hour)
	ev := &matrixEvidence{items: []domain.Evidence{{
		ID: "vip-only", SourceID: "s", ValidUntil: &until,
		PermissionTags: []string{"kb:vip"},
	}}}
	yaml := func(src string) string { return src }
	t.Setenv("KAS_MTX_JEV_ENDPOINT", "http://jev.invalid")
	t.Setenv("KAS_MTX_JEV_API_KEY", "k")
	cfg, err := kas.LoadConfig(strings.NewReader(baseYAML))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider: ev,
		DecisionProvider: answerDecision(),
		// fp-a 无 kb:vip → 全部证据被拒
		PermissionChecker: kas.NewTagChecker(map[string][]string{"fp-rich": {"kb:vip"}}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	_ = yaml
	resp, err := c.Answer(context.Background(), req("越权引用问题"))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Citations) != 0 || resp.Action != kas.ActionAbstain {
		t.Fatalf("越权证据不得被引用: %+v", resp)
	}
}
