package config

import (
	"sort"
	"strconv"
)

// provider/策略参数规则集（Story 2.1.2.2）。provider→阶段映射固定：
//	evidence.search  → first_round（首轮检索；补检轮复用同一 provider 超时，
//	                    由编排层 min(supplement 预算, provider timeout) 保证）
//	jev.decide       → rejudge（复判决策）
//	health.check     → total（独立短超时，不得长于请求根预算）
// 映射依据 docs/design/04 §1（阶段用途表）；变更须先提 ADR。

// ruleEvidenceProvider 校验 evidence 基本参数。
func ruleEvidenceProvider(cfg Config) []Violation {
	var vs []Violation
	if cfg.Evidence.Provider == "" {
		vs = append(vs, newViolation("evidence.provider", "不能为空"))
	}
	if cfg.Evidence.Namespace == "" {
		vs = append(vs, newViolation("evidence.namespace", "不能为空"))
	}
	if cfg.Evidence.TopK <= 0 {
		vs = append(vs, newViolation("evidence.top_k", "必须为正数，当前 %d", cfg.Evidence.TopK))
	}
	if cfg.Evidence.MaxCandidateGroups <= 0 {
		vs = append(vs, newViolation("evidence.max_candidate_groups", "必须为正数，当前 %d", cfg.Evidence.MaxCandidateGroups))
	}
	return vs
}

// ruleProviderTimeouts provider timeout 不得晚于所属阶段预算（边界值相等通过）。
func ruleProviderTimeouts(cfg Config) []Violation {
	var vs []Violation
	if t := cfg.Evidence.Timeout; t != nil && *t > cfg.Timeouts.FirstRound {
		vs = append(vs, newViolation("evidence.timeout",
			"%s 超过所属阶段 first_round 预算 %s", *t, cfg.Timeouts.FirstRound))
	}
	if t := cfg.Decision.Jev.Timeout; t != nil && *t > cfg.Timeouts.Rejudge {
		vs = append(vs, newViolation("decision.jev.timeout",
			"%s 超过所属阶段 rejudge 预算 %s", *t, cfg.Timeouts.Rejudge))
	}
	if cfg.Health.CheckTimeout > cfg.Timeouts.Total {
		vs = append(vs, newViolation("health.check_timeout",
			"%s 超过请求根预算 total %s", cfg.Health.CheckTimeout, cfg.Timeouts.Total))
	}
	return vs
}

// ruleRetryFitsBudget 重试/backoff 最坏耗时必须装入所属阶段预算：
// worst = attempts*timeout + (attempts-1)*backoff。max_attempts=1 时自动满足。
func ruleRetryFitsBudget(cfg Config) []Violation {
	jev := cfg.Decision.Jev
	attempts := jev.MaxAttempts
	if attempts <= 0 {
		attempts = 1 // 默认单次（docs/design/03 §3）
	}
	if attempts == 1 {
		return nil
	}
	timeout := cfg.Timeouts.Rejudge
	if jev.Timeout != nil {
		timeout = *jev.Timeout
	}
	backoff := Duration(0)
	if jev.RetryBackoff != nil {
		backoff = *jev.RetryBackoff
	}
	worst, ok := mulAddOverflow(int64(attempts), int64(timeout), int64(attempts-1), int64(backoff))
	if !ok {
		return []Violation{newViolation("decision.jev.max_attempts", "重试最坏耗时溢出 int64 纳秒域")}
	}
	if Duration(worst) > cfg.Timeouts.Rejudge {
		return []Violation{newViolation("decision.jev.max_attempts",
			"最坏耗时 %s（%d 次 × %s + %d × %s backoff）装不进 rejudge 预算 %s",
			Duration(worst), attempts, timeout, attempts-1, backoff, cfg.Timeouts.Rejudge)}
	}
	return nil
}

// mulAddOverflow 计算 a*b + c*d 并检测溢出。
func mulAddOverflow(a, b, c, d int64) (int64, bool) {
	p1 := a * b
	if a != 0 && p1/a != b {
		return 0, false
	}
	p2 := c * d
	if c != 0 && p2/c != d {
		return 0, false
	}
	sum := p1 + p2
	if (p1 > 0 && p2 > 0 && sum < 0) || (p1 < 0 && p2 < 0 && sum > 0) {
		return 0, false
	}
	return sum, true
}

// ruleValidityPolicy 有效期策略完整性（F-003 前置校验）。
func ruleValidityPolicy(cfg Config) []Violation {
	var vs []Violation
	v := cfg.Evidence.Validity
	if v.RequireValidityPolicyForEachSource && len(v.DefaultMaxAge) == 0 {
		vs = append(vs, newViolation("evidence.validity.default_max_age",
			"require_validity_policy_for_each_source=true 时必须提供至少一个来源策略"))
	}
	if v.FailClosedOnUnknownValidity == nil || !*v.FailClosedOnUnknownValidity {
		vs = append(vs, newViolation("evidence.validity.fail_closed_on_unknown_validity",
			"必须显式为 true（v1.2 fail-closed 安全不变量；有效期未知一律过滤，无放开路径）"))
	}
	for source, age := range v.DefaultMaxAge {
		if source == "" {
			vs = append(vs, newViolation("evidence.validity.default_max_age", "存在空来源 key"))
		}
		if age <= 0 {
			vs = append(vs, newViolation("evidence.validity.default_max_age["+source+"]",
				"max-age 必须为正数，当前 %s", age))
		}
	}
	return vs
}

// cacheCoreKeyDims 是 evidence cache key 的核心维度（缺一不可，F-004）。
var cacheCoreKeyDims = []string{
	"normalized_question", "config_version", "principal_scope_fingerprint",
}

// ruleEvidenceCacheParams 缓存容量与 key 维度校验（启用时）。
func ruleEvidenceCacheParams(cfg Config) []Violation {
	c := cfg.Evidence.Cache
	if !c.Enabled {
		return nil
	}
	var vs []Violation
	if c.Implementation == "" {
		vs = append(vs, newViolation("evidence.cache.implementation", "不能为空"))
	}
	if c.MaxEntries <= 0 {
		vs = append(vs, newViolation("evidence.cache.max_entries", "必须为正数，当前 %d", c.MaxEntries))
	}
	if c.MaxBytes <= 0 {
		vs = append(vs, newViolation("evidence.cache.max_bytes", "必须为正数，当前 %d", c.MaxBytes))
	}
	if c.MaxItemBytes <= 0 {
		vs = append(vs, newViolation("evidence.cache.max_item_bytes", "必须为正数，当前 %d", c.MaxItemBytes))
	}
	if c.MaxItemBytes > c.MaxBytes {
		vs = append(vs, newViolation("evidence.cache.max_item_bytes",
			"%d 超过 max_bytes %d", c.MaxItemBytes, c.MaxBytes))
	}
	if c.TTL <= 0 {
		vs = append(vs, newViolation("evidence.cache.ttl", "必须为正数，当前 %s", c.TTL))
	}
	vs = append(vs, missingKeyDims("evidence.cache.include", c.Include)...)
	return vs
}

// missingKeyDims 检查 include 列表包含全部核心 key 维度。
func missingKeyDims(path string, include []string) []Violation {
	have := make(map[string]bool, len(include))
	for _, k := range include {
		have[k] = true
	}
	var missing []string
	for _, core := range cacheCoreKeyDims {
		if !have[core] {
			missing = append(missing, core)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return []Violation{newViolation(path, "缺少核心 key 维度: %v（F-004 安全要求）", missing)}
	}
	return nil
}

// allowedActions 是规则/action 合法值。
var allowedActions = map[string]bool{
	"answer": true, "supplement": true, "ask_clarification": true, "abstain": true,
}

// ruleDecisionRules 校验降级顺序与规则定义（docs/design/04 §3 顺序固定）。
func ruleDecisionRules(cfg Config) []Violation {
	var vs []Violation
	want := []string{"jev", "deterministic_rules", "decision_cache", "abstain"}
	fb := cfg.Decision.Fallback
	if !equalStrings(fb.Order, want) {
		vs = append(vs, newViolation("decision.fallback.order",
			"降级顺序固定为 %v，当前 %v（docs/design/04 §3）", want, fb.Order))
	}
	seen := map[string]bool{}
	for i, r := range fb.Rules {
		path := "decision.fallback.rules[" + itoa(i) + "]"
		if r.Name == "" {
			vs = append(vs, newViolation(path+".name", "不能为空"))
		}
		if !KnownRuleConditions[r.When] {
			vs = append(vs, newViolation(path+".when", "未知条件 %q（已知: %s/%s）", r.When, CondValidEvidenceZero, CondConfidenceBelowThreshold))
		}
		if r.When == CondConfidenceBelowThreshold {
			if r.Threshold == nil {
				vs = append(vs, newViolation(path+".threshold", "条件 %s 必须配置阈值", CondConfidenceBelowThreshold))
			} else if *r.Threshold < 0 || *r.Threshold > 1 {
				vs = append(vs, newViolation(path+".threshold", "阈值 %v 超出 [0,1]", *r.Threshold))
			}
		}
		if seen[r.Name] {
			vs = append(vs, newViolation(path+".name", "规则名重复: %s", r.Name))
		}
		seen[r.Name] = true
		if r.When == "" {
			vs = append(vs, newViolation(path+".when", "不能为空"))
		}
		if !allowedActions[r.Action] {
			vs = append(vs, newViolation(path+".action", "action %q 不属于有限枚举 answer/supplement/ask_clarification/abstain", r.Action))
		}
	}
	return vs
}

// ruleDecisionCacheParams 受约束决策缓存参数校验（启用时）。
func ruleDecisionCacheParams(cfg Config) []Violation {
	c := cfg.Decision.Fallback.Cache
	if !c.Enabled {
		return nil
	}
	var vs []Violation
	if c.TTL <= 0 {
		vs = append(vs, newViolation("decision.fallback.cache.ttl", "必须为正数，当前 %s", c.TTL))
	}
	if c.MaxEntries <= 0 {
		vs = append(vs, newViolation("decision.fallback.cache.max_entries", "必须为正数，当前 %d", c.MaxEntries))
	}
	if c.ValidateActionAgainstCurrentCandidates {
		// 命中后需再次校验 action 与目标组（docs/design/04 §3），
		// 依赖候选 fingerprint 参与 key，二者必须同时开启。
		if !c.IncludeCandidateFingerprint {
			vs = append(vs, newViolation("decision.fallback.cache.include_candidate_fingerprint",
				"validate_action_against_current_candidates=true 时必须包含候选 fingerprint"))
		}
	}
	return vs
}

// ruleAnswerParams 回答参数校验。
func ruleAnswerParams(cfg Config) []Violation {
	var vs []Violation
	a := cfg.Answer
	if a.MaxCitations <= 0 {
		vs = append(vs, newViolation("answer.max_citations", "必须为正数，当前 %d", a.MaxCitations))
	}
	if a.OutputOnUnknown != "abstain" && a.OutputOnUnknown != "ask_clarification" {
		vs = append(vs, newViolation("answer.output_on_unknown",
			"%q 不合法，仅接受 abstain 或 ask_clarification", a.OutputOnUnknown))
	}
	return vs
}

// ruleHealthParams Health 参数校验。
func ruleHealthParams(cfg Config) []Violation {
	var vs []Violation
	h := cfg.Health
	if h.CheckTimeout <= 0 {
		vs = append(vs, newViolation("health.check_timeout", "必须为正数，当前 %s", h.CheckTimeout))
	}
	known := map[string]bool{"evidence_provider": true, "jev": true}
	for _, c := range h.RequiredComponents {
		if !known[c] {
			vs = append(vs, newViolation("health.required_components", "未知组件 %q（仅支持 evidence_provider/jev）", c))
		}
	}
	for _, c := range h.OptionalComponents {
		if !known[c] {
			vs = append(vs, newViolation("health.optional_components", "未知组件 %q（仅支持 evidence_provider/jev）", c))
		}
	}
	return vs
}

// knownMetricLabels 是允许出现在 label_allowlist 的有界 label 集
// （docs/design/04 §5 表；禁止自由字符串 label）。
var knownMetricLabels = map[string]bool{
	"operation": true, "outcome": true, "action": true, "stage": true,
	"provider": true, "result": true, "reason": true, "route": true,
	"cache_result": true, "fallback_reason": true, "component": true,
	"status": true, "deadline_stage": true, "error_class": true,
}

// ruleMetricsParams 指标配置校验（基数防护）。
func ruleMetricsParams(cfg Config) []Violation {
	if !cfg.Metrics.Enabled {
		return nil
	}
	var vs []Violation
	if cfg.Metrics.Namespace == "" {
		vs = append(vs, newViolation("metrics.namespace", "不能为空"))
	}
	for _, l := range cfg.Metrics.LabelAllowlist {
		if !knownMetricLabels[l] {
			vs = append(vs, newViolation("metrics.label_allowlist",
				"label %q 不在有界 allowlist 中（docs/design/04 §5；禁止 question/principal/Evidence ID/trace ID/config_version/自由字符串）", l))
		}
	}
	return vs
}

// knownTraceFields 是 trace include 白名单（docs/design/02 tracing.include 样例）。
var knownTraceFields = map[string]bool{
	"config_version": true, "evidence_provider": true, "decision_provider": true,
	"model": true, "model_version": true, "fallback_route": true, "cache_result": true,
}

// ruleTracingParams trace 配置校验：正文捕获 v1.2 强制关闭（fail-closed，
// docs/design/01 §6；放开须先提 ADR）。
func ruleTracingParams(cfg Config) []Violation {
	if !cfg.Tracing.Enabled {
		return nil
	}
	var vs []Violation
	if cfg.Tracing.CaptureQuestionText {
		vs = append(vs, newViolation("tracing.capture_question_text",
			"v1.2 强制关闭（fail-closed）；诊断 query 获批准也只用不可逆 hash（docs/design/01 §6）"))
	}
	if cfg.Tracing.CaptureEvidenceText {
		vs = append(vs, newViolation("tracing.capture_evidence_text",
			"v1.2 强制关闭（fail-closed）；Evidence 正文不得进入 trace（docs/design/01 §6）"))
	}
	for _, f := range cfg.Tracing.Include {
		if !knownTraceFields[f] {
			vs = append(vs, newViolation("tracing.include", "字段 %q 不在白名单中", f))
		}
	}
	return vs
}

// equalStrings 逐元素相等。
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// itoa 是 strconv.Itoa 的直连（集中 import）。
func itoa(i int) string { return strconv.Itoa(i) }
