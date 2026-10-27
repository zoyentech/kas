package matrix

import (
	"context"
	"strings"
	"testing"
	"time"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/testutil"
)

// panicProvider 检索即 panic（矩阵行级：经公共 Client 注入）。
type panicProvider struct{}

func (panicProvider) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	panic("matrix: evidence provider exploded")
}

func (panicProvider) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

// TestRowPanicIsolatedThroughClient panic 经 Client 注入：请求降级 abstain、
// 进程存活（Story 8.6.1.1）。
func TestRowPanicIsolatedThroughClient(t *testing.T) {
	c := newClient(t, panicProvider{}, nil, nil)
	resp, err := c.Answer(context.Background(), req("panic 注入问题"))
	if err != nil {
		t.Fatalf("panic 应被隔离并降级: %v", err)
	}
	if resp.Action != kas.ActionAbstain || !resp.Degraded {
		t.Fatalf("应降级 abstain: %+v", resp)
	}
}

// breakerYAML：熔断阈值 2、冷却 1h（测试期不复位），Jev 指向 fake server。
const breakerYAMLTpl = `
schema_version: 1
timeouts: {total: 470ms, preliminary: 40ms, first_round: 120ms, rejudge: 150ms, supplement: 100ms, reserve: 60ms, min_reserve: 40ms}
evidence:
  provider: matrix
  namespace: ns-a
  top_k: 4
  max_candidate_groups: 2
  validity: {require_validity_policy_for_each_source: false, default_max_age: {s: 24h}, fail_closed_on_unknown_validity: true}
  cache: {enabled: false}
decision:
  provider: jev
  jev:
    endpoint_env: KAS_BRK_JEV
    api_key_env: KAS_BRK_KEY
    model: m
    model_version: v1
    max_attempts: 1
    breaker: {enabled: true, failure_threshold: 2, cooldown: 1h}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
balance: {enabled: false}
`

// TestRowBreakerFastFail 熔断 open：零 Jev 调用 + 快速降级 + half-open 恢复
// （Story 8.6.1.1）。
func TestRowBreakerFastFail(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{Status: 500, Body: `{}`})
	defer fake.Close()
	t.Setenv("KAS_BRK_JEV", fake.URL())
	t.Setenv("KAS_BRK_KEY", "k")

	until := time.Now().Add(time.Hour)
	ev := &matrixEvidence{items: []domain.Evidence{{ID: "e1", SourceID: "s", ValidUntil: &until}}}
	cfg, err := kas.LoadConfig(strings.NewReader(breakerYAMLTpl))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: ev})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	// 两次失败（阈值 2）→ open
	for i := 0; i < 2; i++ {
		resp, err := c.Answer(context.Background(), req("熔断问题"))
		if err != nil || !resp.Degraded {
			t.Fatalf("失败请求应降级: %+v %v", resp, err)
		}
	}
	if fake.Calls() != 2 {
		t.Fatalf("熔断前调用数 = %d, want 2", fake.Calls())
	}
	// open 期：快速降级、零 HTTP 调用
	start := time.Now()
	resp, err := c.Answer(context.Background(), req("熔断问题"))
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("open 期应快速降级（远小于 rejudge 150ms）: %s", elapsed)
	}
	if !resp.Degraded {
		t.Fatalf("open 期应降级: %+v", resp)
	}
	if fake.Calls() != 2 {
		t.Fatalf("open 期不得发起 HTTP: %d", fake.Calls())
	}
	// Health detail 可见熔断态
	if h := c.Health(context.Background()); h.Components["jev"].Detail == "" {
		t.Fatal("Health detail 应含熔断状态")
	}
}

// balanceYAML：余额监测开、100ms 轮询（演示快速迁移）。
const balanceYAML = `
schema_version: 1
timeouts: {total: 400ms, preliminary: 40ms, first_round: 120ms, rejudge: 80ms, supplement: 100ms, reserve: 60ms, min_reserve: 40ms}
evidence:
  provider: matrix
  namespace: ns-a
  top_k: 4
  max_candidate_groups: 2
  validity: {require_validity_policy_for_each_source: false, default_max_age: {s: 24h}, fail_closed_on_unknown_validity: true}
  cache: {enabled: false}
decision:
  provider: jev
  jev:
    endpoint_env: KAS_BAL_JEV
    api_key_env: KAS_BAL_KEY
    balance_endpoint_env: KAS_BAL_EP
    model: m
    model_version: v1
    max_attempts: 1
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
balance:
  enabled: true
  interval: 100ms
  low_threshold: 300
  critical_threshold: 50
  sources:
    deepseek: {low_threshold: 100}
`

// waitState 轮询等待余额态达到 want（超时失败）。
func waitState(t *testing.T, c *kas.Client, want kas.BalanceState) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, st := c.Balance(); st == want {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	_, st := c.Balance()
	t.Fatalf("余额态未达 %s（当前 %s）", want, st)
}

// TestRowBalanceLifecycle 余额全链路：fake 端点→poller→Client.Balance/Health
// 四态迁移（Story 8.6.1.1）。
func TestRowBalanceLifecycle(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{Status: 200, Body: testutil.ValidJevBody("answer", "g1", 0.9)})
	fake.SetBalanceAmount(500)
	defer fake.Close()
	t.Setenv("KAS_BAL_JEV", fake.URL())
	t.Setenv("KAS_BAL_KEY", "k")
	t.Setenv("KAS_BAL_EP", fake.URL()+"/balance")

	until := time.Now().Add(time.Hour)
	ev := &matrixEvidence{items: []domain.Evidence{{ID: "e1", SourceID: "s", ValidUntil: &until}}}
	cfg, err := kas.LoadConfig(strings.NewReader(balanceYAML))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: ev})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	// normal → low → critical
	waitState(t, c, kas.BalanceNormal)
	fake.SetBalanceAmount(200)
	waitState(t, c, kas.BalanceLow)
	fake.SetBalanceAmount(10)
	waitState(t, c, kas.BalanceCritical)

	info, st := c.Balance()
	if st != kas.BalanceCritical || info.Amount != 10 || info.Unit != "credit" {
		t.Fatalf("快照异常: %+v %s", info, st)
	}
	// Health 信息组件
	h := c.Health(context.Background())
	bc, ok := h.Components["balance:jev"]
	if !ok || bc.Status != kas.ComponentNotReady {
		t.Fatalf("critical 余额应反映到 Health: %+v", h.Components)
	}
}

// TestRowMultiSourceBalance 多源余额 + 热更新 API（v1.4）：
// jev 内置源 + 运行时 Upsert deepseek 源 → 各自快照/Health/gauge 事件分流。
func TestRowMultiSourceBalance(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{Status: 200, Body: testutil.ValidJevBody("answer", "g1", 0.9)})
	fake.SetBalanceAmount(500)
	defer fake.Close()
	t.Setenv("KAS_BAL_JEV", fake.URL())
	t.Setenv("KAS_BAL_KEY", "k")
	t.Setenv("KAS_BAL_EP", fake.URL()+"/balance")
	// deepseek 内置源条件：DEEPSEEK_TOKEN 指向本地 fake
	dsFake := testutil.NewJevFakeServer(testutil.JevScript{BalanceBody: `{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"120.00"}]}`})
	defer dsFake.Close()
	t.Setenv("DEEPSEEK_TOKEN", "ds-key")
	t.Setenv("DEEPSEEK_BALANCE_ENDPOINT", dsFake.URL()+"/balance")
	// qwen 内置源条件：QWEN_APIKEY 在用 + 阿里云 BSS 凭据指向本地 fake
	// BSS 适配器 GET 根路径（带签名 query），fake 的默认分支按 Body 应答
	bssJSON := `{"Success":true,"Data":{"AvailableCashAmount":"88.66","Currency":"CNY"}}`
	aliFake := testutil.NewJevFakeServer(testutil.JevScript{Body: bssJSON, BalanceBody: bssJSON})
	defer aliFake.Close()
	t.Setenv("QWEN_APIKEY", "qwen-key")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "ak")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "sk")
	t.Setenv("ALIBABA_CLOUD_BSS_ENDPOINT", aliFake.URL())

	until := time.Now().Add(time.Hour)
	ev := &matrixEvidence{items: []domain.Evidence{{ID: "e1", SourceID: "s", ValidUntil: &until}}}
	cfg, err := kas.LoadConfig(strings.NewReader(balanceYAML))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: ev})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	// 内置源自动注册：jev + deepseek + aliyun + qwen（QWEN_APIKEY 标识在用）
	sources := c.BalanceSources()
	if len(sources) != 4 {
		t.Fatalf("应自动注册 jev+deepseek+aliyun+qwen: %v", sources)
	}
	waitState(t, c, kas.BalanceNormal) // 默认源 jev=500
	waitSourceState(t, c, "deepseek", kas.BalanceNormal)
	waitSourceState(t, c, "qwen", kas.BalanceLow) // 88.66 ≤ 300（全局阈值）
	waitSourceState(t, c, "aliyun", kas.BalanceLow)
	// qwen 与 aliyun 同为账户现金余额：断言金额一致（组件视图复用计费源）
	qi, qst, ok := c.BalanceOf("qwen")
	if !ok || qst != kas.BalanceLow || qi.Amount != 88.66 || qi.Unit != "CNY" {
		t.Fatalf("qwen 源异常: %+v %s %v", qi, qst, ok)
	}
	if ai, _, ok := c.BalanceOf("aliyun"); !ok || ai.Amount != qi.Amount {
		t.Fatalf("aliyun 与 qwen 金额应一致（同一 BSS 计费源）: %+v vs %+v", ai, qi)
	}

	// Health 按源分列
	h := c.Health(context.Background())
	if _, ok := h.Components["balance:jev"]; !ok {
		t.Fatal("Health 应含 balance:jev")
	}
	if _, ok := h.Components["balance:deepseek"]; !ok {
		t.Fatal("Health 应含 balance:deepseek")
	}
	if _, ok := h.Components["balance:qwen"]; !ok {
		t.Fatal("Health 应含 balance:qwen")
	}

	// 热更新：运行时新增 host 源（模拟宿主管理后台注入自定义计费源）
	hostSrc := staticBalance{amount: 5}
	if err := c.UpsertBalanceSource("host", hostSrc, cfg.Balance); err != nil {
		t.Fatal(err)
	}
	if got := c.BalanceSources(); len(got) != 5 {
		t.Fatalf("热更新后应 5 源: %v", got)
	}
	waitSourceState(t, c, "host", kas.BalanceCritical) // 5 ≤ 50
	// 移除源
	c.RemoveBalanceSource("host")
	if got := c.BalanceSources(); len(got) != 4 {
		t.Fatalf("移除后应 4 源: %v", got)
	}
}

// staticBalance 固定金额的宿主注入源。
type staticBalance struct{ amount float64 }

func (s staticBalance) Balance(ctx context.Context) (kas.BalanceInfo, error) {
	return kas.BalanceInfo{Source: "host", Amount: s.amount, Unit: "CNY"}, nil
}

// waitSourceState 等待指定源达到目标态。
func waitSourceState(t *testing.T, c *kas.Client, name string, want kas.BalanceState) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, st, ok := c.BalanceOf(name); ok && st == want {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	_, st, _ := c.BalanceOf(name)
	t.Fatalf("源 %s 未达 %s（当前 %s）", name, want, st)
}
