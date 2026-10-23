package config

import (
	"testing"
	"time"
)

// TestProviderTimeoutWithinStage provider timeout ≤ 阶段预算；边界相等通过
// （Story 2.1.2.2 T1）。
func TestProviderTimeoutWithinStage(t *testing.T) {
	cfg := validConfig()
	// evidence.timeout 超过 first_round：拒绝
	t1 := D(1800*time.Millisecond + 1)
	cfg.Evidence.Timeout = &t1
	if err := Validate(cfg); err == nil || !contains(err.Error(), "evidence.timeout") {
		t.Fatalf("evidence timeout 超 first_round 应拒绝，实际: %v", err)
	}
	// 边界相等：通过
	t2 := D(1800 * time.Millisecond)
	cfg.Evidence.Timeout = &t2
	if err := Validate(cfg); err != nil {
		t.Fatalf("evidence timeout 等于 first_round 应通过，实际: %v", err)
	}
	// jev.timeout 超 rejudge：拒绝
	j1 := D(900*time.Millisecond + 1)
	cfg.Decision.Jev.Timeout = &j1
	if err := Validate(cfg); err == nil || !contains(err.Error(), "decision.jev.timeout") {
		t.Fatalf("jev timeout 超 rejudge 应拒绝，实际: %v", err)
	}
	// health.check_timeout 超 total：拒绝
	cfg.Health.CheckTimeout = D(4801 * time.Millisecond)
	if err := Validate(cfg); err == nil || !contains(err.Error(), "health.check_timeout") {
		t.Fatalf("health check_timeout 超 total 应拒绝，实际: %v", err)
	}
}

// TestRetryFitsBudget 重试最坏耗时装入阶段预算（Story 2.1.2.2 T2）。
func TestRetryFitsBudget(t *testing.T) {
	// max_attempts=1（默认）：无需 timeout 配置即通过
	cfg := validConfig()
	if err := Validate(cfg); err != nil {
		t.Fatalf("max_attempts=1 应通过: %v", err)
	}
	// 3 次 × 300ms + 2 × 50ms = 1000ms > 900ms rejudge：拒绝
	cfg.Decision.Jev.MaxAttempts = 3
	to := D(300 * time.Millisecond)
	bo := D(50 * time.Millisecond)
	cfg.Decision.Jev.Timeout = &to
	cfg.Decision.Jev.RetryBackoff = &bo
	if err := Validate(cfg); err == nil || !contains(err.Error(), "max_attempts") {
		t.Fatalf("重试装不进预算应拒绝，实际: %v", err)
	}
	// 3 次 × 250ms + 2 × 50ms = 850ms ≤ 900ms：通过
	to2 := D(250 * time.Millisecond)
	cfg.Decision.Jev.Timeout = &to2
	if err := Validate(cfg); err != nil {
		t.Fatalf("重试装入预算应通过，实际: %v", err)
	}
}

// TestValidityPolicyCompleteness 有效期策略完整性（Story 2.1.2.2 T3）。
func TestValidityPolicyCompleteness(t *testing.T) {
	cfg := validConfig()
	cfg.Evidence.Validity.DefaultMaxAge = nil
	if err := Validate(cfg); err == nil || !contains(err.Error(), "default_max_age") {
		t.Fatalf("require=true 且策略为空应拒绝，实际: %v", err)
	}
	cfg.Evidence.Validity.DefaultMaxAge = map[string]Duration{"product_manual": D(-1 * time.Hour)}
	if err := Validate(cfg); err == nil || !contains(err.Error(), "product_manual") {
		t.Fatalf("非法 max-age 应拒绝，实际: %v", err)
	}
	// docs/02 两种来源样例通过（validConfig 基线即含 product_manual/service_notice）
	cfg.Evidence.Validity.DefaultMaxAge = map[string]Duration{
		"product_manual": D(720 * time.Hour), "service_notice": D(24 * time.Hour),
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("docs/02 来源样例应通过，实际: %v", err)
	}
}

// TestEvidenceCacheParams 缓存容量参数校验（Story 2.1.2.2 T4）。
func TestEvidenceCacheParams(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		path   string
	}{
		{"max_entries_zero", func(c *Config) { c.Evidence.Cache.MaxEntries = 0 }, "evidence.cache.max_entries"},
		{"max_bytes_negative", func(c *Config) { c.Evidence.Cache.MaxBytes = -1 }, "evidence.cache.max_bytes"},
		{"item_gt_total", func(c *Config) { c.Evidence.Cache.MaxItemBytes = c.Evidence.Cache.MaxBytes + 1 }, "evidence.cache.max_item_bytes"},
		{"ttl_zero", func(c *Config) { c.Evidence.Cache.TTL = 0 }, "evidence.cache.ttl"},
		{"missing_core_dim", func(c *Config) {
			c.Evidence.Cache.Include = []string{"config_version", "principal_scope_fingerprint"}
		}, "evidence.cache.include"},
	}
	for _, tc := range tests {
		cfg := validConfig()
		tc.mutate(&cfg)
		if err := Validate(cfg); err == nil || !contains(err.Error(), tc.path) {
			t.Errorf("%s: 应拒绝且定位 %s，实际: %v", tc.name, tc.path, err)
		}
	}
	// 禁用时参数不校验
	cfg := validConfig()
	cfg.Evidence.Cache.Enabled = false
	cfg.Evidence.Cache.MaxEntries = 0
	if err := Validate(cfg); err != nil {
		t.Fatalf("禁用时不应校验容量，实际: %v", err)
	}
}

// TestDecisionRules 降级顺序固定与规则 action 枚举。
func TestDecisionRules(t *testing.T) {
	cfg := validConfig()
	cfg.Decision.Fallback.Order = []string{"jev", "decision_cache", "deterministic_rules", "abstain"}
	if err := Validate(cfg); err == nil || !contains(err.Error(), "decision.fallback.order") {
		t.Fatalf("非固定降级顺序应拒绝，实际: %v", err)
	}
	cfg = validConfig()
	cfg.Decision.Fallback.Rules[0].Action = "do_nothing"
	if err := Validate(cfg); err == nil || !contains(err.Error(), "rules[0].action") {
		t.Fatalf("非法 action 应拒绝，实际: %v", err)
	}
}

// TestDecisionCacheParams 受约束决策缓存组合合法性。
func TestDecisionCacheParams(t *testing.T) {
	cfg := validConfig()
	cfg.Decision.Fallback.Cache.IncludeCandidateFingerprint = false
	if err := Validate(cfg); err == nil || !contains(err.Error(), "include_candidate_fingerprint") {
		t.Fatalf("validate_action=true 而 candidate fingerprint 缺失应拒绝，实际: %v", err)
	}
}

// TestAnswerParams answer 参数校验。
func TestAnswerParams(t *testing.T) {
	cfg := validConfig()
	cfg.Answer.OutputOnUnknown = "guess"
	if err := Validate(cfg); err == nil || !contains(err.Error(), "output_on_unknown") {
		t.Fatalf("非法 output_on_unknown 应拒绝，实际: %v", err)
	}
	cfg.Answer.MaxCitations = 0
	if err := Validate(cfg); err == nil || !contains(err.Error(), "answer.max_citations") {
		t.Fatalf("max_citations=0 应拒绝，实际: %v", err)
	}
}

// TestHealthParams health 组件名合法。
func TestHealthParams(t *testing.T) {
	cfg := validConfig()
	cfg.Health.RequiredComponents = []string{"redis"}
	if err := Validate(cfg); err == nil || !contains(err.Error(), "required_components") {
		t.Fatalf("未知组件应拒绝，实际: %v", err)
	}
}

// TestMetricsLabelAllowlist label 白名单（基数防护）。
func TestMetricsLabelAllowlist(t *testing.T) {
	cfg := validConfig()
	cfg.Metrics.LabelAllowlist = []string{"operation", "question"} // question 禁止
	if err := Validate(cfg); err == nil || !contains(err.Error(), "question") {
		t.Fatalf("question 作为 label 应拒绝，实际: %v", err)
	}
	cfg.Metrics.LabelAllowlist = []string{"operation", "tenant_key"} // 身份明文禁止
	if err := Validate(cfg); err == nil {
		t.Fatal("tenant_key 作为 label 应拒绝")
	}
}

// TestTracingFailClosed 正文捕获 v1.2 强制关闭。
func TestTracingFailClosed(t *testing.T) {
	cfg := validConfig()
	cfg.Tracing.CaptureQuestionText = true
	if err := Validate(cfg); err == nil || !contains(err.Error(), "capture_question_text") {
		t.Fatalf("capture_question_text=true 应拒绝（fail-closed），实际: %v", err)
	}
	cfg = validConfig()
	cfg.Tracing.Include = append(cfg.Tracing.Include, "question_text")
	if err := Validate(cfg); err == nil || !contains(err.Error(), "question_text") {
		t.Fatalf("include 白名单外字段应拒绝，实际: %v", err)
	}
}

// TestValidationBranchSweep 补齐剩余分支：空/重复规则名、implementation 空、
// 禁用分支、未知 include 字段集合语义。
func TestValidationBranchSweep(t *testing.T) {
	// 规则 name/when 空与重复
	cfg := validConfig()
	cfg.Decision.Fallback.Rules = []RuleConfig{
		{Name: "r", When: "", Action: "abstain"},
		{Name: "r", When: "valid_evidence_count_equals_zero", Action: "abstain"},
		{Name: "", When: "x", Action: "abstain"},
	}
	err := Validate(cfg)
	if err == nil || !contains(err.Error(), "rules[0].when") || !contains(err.Error(), "rules[1].name") || !contains(err.Error(), "rules[2].name") {
		t.Fatalf("规则字段校验缺失，实际: %v", err)
	}
	// cache implementation 空
	cfg = validConfig()
	cfg.Evidence.Cache.Implementation = ""
	if err := Validate(cfg); err == nil || !contains(err.Error(), "evidence.cache.implementation") {
		t.Fatalf("implementation 空应拒绝，实际: %v", err)
	}
	// decision cache 禁用：非法参数不校验
	cfg = validConfig()
	cfg.Decision.Fallback.Cache.Enabled = false
	cfg.Decision.Fallback.Cache.MaxEntries = -1
	if err := Validate(cfg); err != nil {
		t.Fatalf("decision cache 禁用不应校验，实际: %v", err)
	}
	// metrics 禁用：namespace 空不校验
	cfg = validConfig()
	cfg.Metrics.Enabled = false
	cfg.Metrics.Namespace = ""
	if err := Validate(cfg); err != nil {
		t.Fatalf("metrics 禁用不应校验 namespace，实际: %v", err)
	}
	// tracing 禁用：正文捕获不校验
	cfg = validConfig()
	cfg.Tracing.Enabled = false
	cfg.Tracing.CaptureQuestionText = true
	cfg.Tracing.CaptureEvidenceText = true
	if err := Validate(cfg); err != nil {
		t.Fatalf("tracing 禁用不应校验正文开关，实际: %v", err)
	}
	// evidence namespace 空
	cfg = validConfig()
	cfg.Evidence.Namespace = ""
	if err := Validate(cfg); err == nil || !contains(err.Error(), "evidence.namespace") {
		t.Fatalf("namespace 空应拒绝，实际: %v", err)
	}
	// 空来源 key
	cfg = validConfig()
	cfg.Evidence.Validity.DefaultMaxAge[""] = D(time.Hour)
	if err := Validate(cfg); err == nil || !contains(err.Error(), "空来源") {
		t.Fatalf("空来源 key 应拒绝，实际: %v", err)
	}
	// jev attempts 溢出构造
	cfg = validConfig()
	cfg.Decision.Jev.MaxAttempts = 1 << 40
	to := D(900 * time.Millisecond)
	cfg.Decision.Jev.Timeout = &to
	if err := Validate(cfg); err == nil {
		t.Fatal("attempts 溢出应拒绝")
	}
}
