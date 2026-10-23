package kas_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kas "github.com/kas-runtime/kas"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// stubEvidenceProvider 满足接口的最小实现（S4 前仅用于装配校验）。
type stubEvidenceProvider struct{}

func (stubEvidenceProvider) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	return kas.EvidenceResult{}, kas.ErrEvidenceUnavailable
}

func (stubEvidenceProvider) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{}
}

const validYAML = `
schema_version: 1
timeouts: {total: 4800ms, preliminary: 400ms, first_round: 1800ms, rejudge: 900ms, supplement: 1100ms, reserve: 600ms}
evidence:
  provider: host-search
  namespace: support-public
  top_k: 8
  max_candidate_groups: 4
  validity: {require_validity_policy_for_each_source: true, default_max_age: {product_manual: 720h}}
  cache: {enabled: false}
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

func mustClient(t *testing.T) *kas.Client {
	t.Helper()
	cfg, err := kas.LoadConfig(strings.NewReader(validYAML))
	if err != nil {
		t.Fatalf("加载基线配置失败: %v", err)
	}
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: stubEvidenceProvider{}})
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestNewRequiresEvidenceProvider 缺必需依赖报 ErrInvalidConfig
// （Story 2.2.2.1 T2）。
func TestNewRequiresEvidenceProvider(t *testing.T) {
	cfg, _ := kas.LoadConfig(strings.NewReader(validYAML))
	_, err := kas.New(cfg, kas.Dependencies{})
	if err == nil || !errors.Is(err, kas.ErrInvalidConfig) {
		t.Fatalf("缺 EvidenceProvider 应报 ErrInvalidConfig，实际: %v", err)
	}
	if !strings.Contains(err.Error(), "EvidenceProvider") {
		t.Fatalf("错误应指明缺失依赖，实际: %v", err)
	}
}

// TestNewRejectsInvalidConfig New 执行 New 级同步校验（docs/03 §1）。
func TestNewRejectsInvalidConfig(t *testing.T) {
	bad := strings.Replace(validYAML, "total: 4800ms", "total: 100ms", 1)
	if _, err := kas.LoadConfig(strings.NewReader(bad)); err == nil {
		t.Fatal("配置本身非法，LoadConfig 应先失败")
	}
	// 直接构造非法 Config 绕过 Load 的校验，验证 New 兜底
	var broken kas.Config
	if _, err := kas.New(broken, kas.Dependencies{EvidenceProvider: stubEvidenceProvider{}}); err == nil {
		t.Fatal("New 必须拒绝非法配置（双重防线）")
	}
}

// TestReloadAtomicity 通过公共 API：失败保持旧版本、成功返回新版本（F-011）。
func TestReloadAtomicity(t *testing.T) {
	c := mustClient(t)
	v0 := c.ConfigVersion()

	// 失败 Reload
	bad := strings.Replace(validYAML, "reserve: 600ms", "reserve: 1ms", 1)
	if _, err := c.Reload(context.Background(), strings.NewReader(bad)); err == nil {
		t.Fatal("无效 Reload 应失败")
	} else if !errors.Is(err, kas.ErrInvalidConfig) {
		t.Fatalf("Reload 错误应分类 ErrInvalidConfig，实际: %v", err)
	}
	if c.ConfigVersion() != v0 {
		t.Fatalf("失败 Reload 后版本变化: %s -> %s", v0, c.ConfigVersion())
	}

	// 成功 Reload（top_k 8→10 产生新版本）
	mutated := strings.Replace(validYAML, "top_k: 8", "top_k: 10", 1)
	newV, err := c.Reload(context.Background(), strings.NewReader(mutated))
	if err != nil {
		t.Fatalf("合法 Reload 应成功: %v", err)
	}
	if newV == v0 {
		t.Fatal("Reload 成功应产生新版本")
	}
	if c.ConfigVersion() != newV {
		t.Fatalf("Reload 返回版本与当前不一致: %s vs %s", newV, c.ConfigVersion())
	}
}

// TestClosedClient Close 后 Answer/Reload 返回 ErrClosed。
func TestClosedClient(t *testing.T) {
	cfg, _ := kas.LoadConfig(strings.NewReader(validYAML))
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: stubEvidenceProvider{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Answer(context.Background(), kas.Request{}); !errors.Is(err, kas.ErrClosed) {
		t.Fatalf("Close 后 Answer 应返回 ErrClosed，实际: %v", err)
	}
	if _, err := c.Reload(context.Background(), strings.NewReader(validYAML)); !errors.Is(err, kas.ErrClosed) {
		t.Fatalf("Close 后 Reload 应返回 ErrClosed，实际: %v", err)
	}
}
