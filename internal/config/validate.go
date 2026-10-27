package config

// Validate 执行全部启动校验（Story 2.1.1.2 T2 规则框架 + 各规则集）。
// 聚合返回全部违规而非首条；规则清单映射 docs/design/04 §1"拒绝"项：
//
//	拒绝：预算为非正数（validate_timeouts.go）
//	     duration 加法溢出（validate_timeouts.go）
//	     total 与五段之和不等（validate_timeouts.go，F-001）
//	     total 超宿主上限（validate_timeouts.go，O-4 host_max_total）
//	     reserve 小于经评审的最小组装预算（validate_timeouts.go，O-5 默认 200ms）
//	     provider timeout 超阶段预算（validate_providers.go）
//	     重试/backoff 无法装入阶段预算（validate_providers.go）
//	     未知配置字段或 duration 格式错误（decode.go / duration.go）
//	     有效期策略缺失/非法（validate_providers.go）
//	     缓存容量参数非法（validate_providers.go）
func Validate(cfg Config) error {
	var vs []Violation
	for _, rule := range validationRules {
		vs = append(vs, rule(cfg)...)
	}
	if len(vs) > 0 {
		return &ValidationError{Violations: vs}
	}
	return nil
}

// validationRules 是有序规则表；顺序即错误输出顺序（稳定可测）。
var validationRules = []func(Config) []Violation{
	ruleSchemaVersion,
	ruleTimeoutsPositive,
	ruleTimeoutsOverflow,
	ruleTimeoutsEquation,
	ruleHostMaxTotal,
	ruleMinReserve,
	ruleEvidenceProvider,
	ruleProviderTimeouts,
	ruleRetryFitsBudget,
	ruleValidityPolicy,
	ruleEvidenceCacheParams,
	ruleDecisionRules,
	ruleDecisionCacheParams,
	ruleAnswerParams,
	ruleHealthParams,
	ruleMetricsParams,
	ruleTracingParams,
	ruleBalanceConfig,
	ruleJevBreaker,
}

// ruleSchemaVersion 校验 schema 版本（v1.2 仅接受 1）。
func ruleSchemaVersion(cfg Config) []Violation {
	if cfg.SchemaVersion != 1 {
		return []Violation{newViolation("schema_version", "版本 %d 不受支持，v1.2 仅接受 1", cfg.SchemaVersion)}
	}
	return nil
}
