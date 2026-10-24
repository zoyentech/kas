package runtime_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/runtime"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

const validYAML = `
schema_version: 1
timeouts: {total: 4800ms, preliminary: 400ms, first_round: 1800ms, rejudge: 900ms, supplement: 1100ms, reserve: 600ms}
evidence:
  provider: host-search
  namespace: support-public
  top_k: 8
  max_candidate_groups: 4
  validity:
    require_validity_policy_for_each_source: true
    fail_closed_on_unknown_validity: true
    default_max_age: {product_manual: 720h, service_notice: 24h}
  cache:
    enabled: false
decision:
  provider: jev
  jev: {endpoint_env: KAS_JEV_ENDPOINT, api_key_env: KAS_JEV_API_KEY, model: jev-latest, model_version: stable, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: r1, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 500}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
`

// mutatedYAML 返回 top_k 不同的另一份合法配置。
const mutatedYAML = `
schema_version: 1
timeouts: {total: 4800ms, preliminary: 400ms, first_round: 1800ms, rejudge: 900ms, supplement: 1100ms, reserve: 600ms}
evidence:
  provider: host-search
  namespace: support-public
  top_k: 10
  max_candidate_groups: 4
  validity:
    require_validity_policy_for_each_source: true
    fail_closed_on_unknown_validity: true
    default_max_age: {product_manual: 720h, service_notice: 24h}
  cache:
    enabled: false
decision:
  provider: jev
  jev: {endpoint_env: KAS_JEV_ENDPOINT, api_key_env: KAS_JEV_API_KEY, model: jev-latest, model_version: stable, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: r1, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 500}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
`

// TestBuildSnapshotRejectsInvalid 校验失败返回 ErrInvalidConfig 且不产生快照
// （绕过 LoadConfig 前置校验，程序化构造非法 Config 专测 Build 兜底）。
func TestBuildSnapshotRejectsInvalid(t *testing.T) {
	cfg := mustLoad(t, validYAML)
	cfg.Timeouts.Total = cfg.Timeouts.Total + 1 // 破坏预算等式
	_, err := runtime.BuildSnapshot(cfg)
	if err == nil {
		t.Fatal("非法配置构建快照必须失败")
	}
	if !strings.Contains(err.Error(), "timeouts.total") {
		t.Fatalf("错误应含字段路径，实际: %v", err)
	}
}

func TestSnapshotImmutableAccessors(t *testing.T) {
	snap := mustBuild(t, validYAML)
	if snap.Version() == "" {
		t.Fatal("版本非空")
	}
	if snap.Budget().Total.Std().Milliseconds() != 4800 {
		t.Fatalf("Budget Total = %s", snap.Budget().Total)
	}
	if snap.BuiltAt().IsZero() {
		t.Fatal("BuiltAt 应记录")
	}
	if snap.Config().Evidence.TopK != 8 {
		t.Fatalf("Config 访问器异常: %+v", snap.Config().Evidence)
	}
}

// TestManagerReplaceAndLoad 单线程替换/读取正确性（Story 2.2.2.2 T1）。
func TestManagerReplaceAndLoad(t *testing.T) {
	m := runtime.NewManager(mustBuild(t, validYAML))
	if m.Version() == "" {
		t.Fatal("初始版本非空")
	}
	old := m.Current()
	snap2 := mustBuild(t, mutatedYAML)
	m.Replace(snap2)
	if m.Current().Version() != snap2.Version() {
		t.Fatal("替换后 Current 应为新快照")
	}
	if m.Version() != snap2.Version() {
		t.Fatal("Version 应反映新快照")
	}
	if old.Version() == snap2.Version() {
		t.Fatal("两份配置版本应不同")
	}
}

// TestReloadFailureKeepsCurrent 无效 Reload：错误返回且当前版本完全不变
// （Story 2.2.2.2 T2，F-011）。
func TestReloadFailureKeepsCurrent(t *testing.T) {
	m := runtime.NewManager(mustBuild(t, validYAML))
	before := m.Version()

	bad := strings.Replace(validYAML, "reserve: 600ms", "reserve: 601ms", 1)
	_, err := runtime.Reload(context.Background(), m, strings.NewReader(bad))
	if err == nil {
		t.Fatal("破坏预算等式的 Reload 必须失败")
	}
	if m.Version() != before {
		t.Fatalf("失败 Reload 后版本变化: %s -> %s", before, m.Version())
	}
	if m.Current().Config().Evidence.TopK != 8 {
		t.Fatal("失败 Reload 后配置被污染")
	}

	// 未知字段同样拒绝
	unknown := strings.Replace(validYAML, "top_k: 8", "topk: 8", 1)
	if _, err := runtime.Reload(context.Background(), m, strings.NewReader(unknown)); err == nil {
		t.Fatal("未知字段 Reload 必须失败")
	}
	if m.Version() != before {
		t.Fatal("未知字段 Reload 失败后版本变化")
	}
}

// TestReloadSuccess 返回新版本并原子替换。
func TestReloadSuccess(t *testing.T) {
	m := runtime.NewManager(mustBuild(t, validYAML))
	res, err := runtime.Reload(context.Background(), m, strings.NewReader(mutatedYAML))
	if err != nil {
		t.Fatalf("合法 Reload 应成功: %v", err)
	}
	if res.Version != m.Version() || res.Snapshot.Version() != res.Version {
		t.Fatalf("Reload 结果与 current 不一致: %+v vs %s", res, m.Version())
	}
}

func mustBuild(t *testing.T, yamlSrc string) *runtime.Snapshot {
	t.Helper()
	snap, err := runtime.BuildSnapshot(mustLoad(t, yamlSrc))
	if err != nil {
		t.Fatalf("构建快照失败: %v", err)
	}
	return snap
}

func mustLoad(t *testing.T, yamlSrc string) config.Config {
	t.Helper()
	c, err := config.LoadConfig(strings.NewReader(yamlSrc))
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	return c
}

// TestReloadEdgeCases nil manager 与已取消 ctx。
func TestReloadEdgeCases(t *testing.T) {
	if _, err := runtime.Reload(context.Background(), nil, strings.NewReader(validYAML)); err == nil {
		t.Fatal("nil manager 应报错")
	}
	m := runtime.NewManager(mustBuild(t, validYAML))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runtime.Reload(ctx, m, strings.NewReader(validYAML)); err == nil {
		t.Fatal("已取消 ctx 应报错")
	}
	if m.Current().Version() == "" {
		t.Fatal("取消的 Reload 不应影响 current")
	}
}
