package config

// Config 是 v1.2 配置模型，与 docs/design/02 示例 YAML 逐节对应（Story 2.1.1.1）。
// YAML 严格解析（未知字段拒绝）；secret 明文不允许出现在任何字段，密钥由
// 宿主 secret manager / 环境变量注入，配置只含 *_env 引用名。
type Config struct {
	// SchemaVersion 是配置 schema 版本；v1.2 仅接受 1。
	SchemaVersion int            `yaml:"schema_version" json:"schema_version"`
	Timeouts      TimeoutBudget  `yaml:"timeouts" json:"timeouts"`
	Evidence      EvidenceConfig `yaml:"evidence" json:"evidence"`
	Decision      DecisionConfig `yaml:"decision" json:"decision"`
	Answer        AnswerConfig   `yaml:"answer" json:"answer"`
	Health        HealthConfig   `yaml:"health" json:"health"`
	Metrics       MetricsConfig  `yaml:"metrics" json:"metrics"`
	Tracing       TracingConfig  `yaml:"tracing" json:"tracing"`
	// Balance 余额监测配置（v1.3 F-016；零值 = 禁用）。
	Balance BalanceConfig `yaml:"balance" json:"balance"`
}

// BalanceConfig 是余额监测配置（Story 8.2.1.1；v1.4 多源扩展）。
type BalanceConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Interval Duration `yaml:"interval" json:"interval"`
	// LowThreshold 低余额阈值（amount ≤ low 且 > critical 记 low）——全局默认。
	LowThreshold float64 `yaml:"low_threshold" json:"low_threshold"`
	// CriticalThreshold 危急阈值（amount ≤ critical 记 critical）——全局默认。
	CriticalThreshold float64 `yaml:"critical_threshold" json:"critical_threshold"`
	// Sources 按源名覆盖全局默认（v1.4，F-016 多源；未列出的源用全局默认）。
	// 源名为内置标识（jev/deepseek/aliyun）或宿主 UpsertBalanceSource 的名字。
	Sources map[string]BalanceSourceConfig `yaml:"sources,omitempty" json:"sources,omitempty"`
}

// BalanceSourceConfig 是单源覆盖项（零值字段回退全局默认）。
type BalanceSourceConfig struct {
	Interval          Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	LowThreshold      *float64 `yaml:"low_threshold,omitempty" json:"low_threshold,omitempty"`
	CriticalThreshold *float64 `yaml:"critical_threshold,omitempty" json:"critical_threshold,omitempty"`
}

// TimeoutBudget 是五阶段请求预算；Total 必须精确等于其余五段之和（F-001）。
type TimeoutBudget struct {
	// Total 请求总预算：root deadline = min(caller deadline, start+Total)。
	Total Duration `yaml:"total" json:"total"`
	// Preliminary 初判：输入范围、澄清与检索路径。
	Preliminary Duration `yaml:"preliminary" json:"preliminary"`
	// FirstRound 首轮：cache/检索、候选组。
	FirstRound Duration `yaml:"first_round" json:"first_round"`
	// Rejudge 复判：对首轮有效候选做决策。
	Rejudge Duration `yaml:"rejudge" json:"rejudge"`
	// Supplement 补检：额外检索与最终校验。
	Supplement Duration `yaml:"supplement" json:"supplement"`
	// Reserve 余量：引用检查、组装、编码。
	Reserve Duration `yaml:"reserve" json:"reserve"`
	// HostMaxTotal 宿主 total 预算上限（O-4）；未配置（nil）则不校验。
	HostMaxTotal *Duration `yaml:"host_max_total,omitempty" json:"host_max_total,omitempty"`
	// MinReserve reserve 最小组装预算下限（O-5）；未配置时默认 200ms。
	MinReserve *Duration `yaml:"min_reserve,omitempty" json:"min_reserve,omitempty"`
}

// Stages 返回五段预算（不含 Total），顺序固定。
func (b TimeoutBudget) Stages() []Duration {
	return []Duration{b.Preliminary, b.FirstRound, b.Rejudge, b.Supplement, b.Reserve}
}

// Sum 返回五段之和；溢出时第二个返回值为 false（由校验层拒绝）。
func (b TimeoutBudget) Sum() (Duration, bool) {
	var sum int64
	for _, s := range b.Stages() {
		v := int64(s)
		if v > 0 && sum > int64(1<<63-1)-v {
			return 0, false
		}
		sum += v
	}
	return Duration(sum), true
}

// EvidenceConfig 是 Evidence 链路配置（provider、检索参数、有效期策略与缓存）。
type EvidenceConfig struct {
	// Provider EvidenceProvider 实现标识（v1.2 内置 host-search 语义由宿主注入实现）。
	Provider           string              `yaml:"provider" json:"provider"`
	Namespace          string              `yaml:"namespace" json:"namespace"`
	TopK               int                 `yaml:"top_k" json:"top_k"`
	MaxCandidateGroups int                 `yaml:"max_candidate_groups" json:"max_candidate_groups"`
	Validity           ValidityConfig      `yaml:"validity" json:"validity"`
	Cache              EvidenceCacheConfig `yaml:"cache" json:"cache"`
	// Timeout provider 调用超时（可选）；未配置时默认 FirstRound 预算，
	// 校验要求 ≤ FirstRound（docs/design/04 §1 provider timeout 超阶段预算拒绝）。
	Timeout *Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// ValidityConfig 是知识有效期策略（F-003）。
type ValidityConfig struct {
	// RequireValidityPolicyForEachSource 为 true 时要求每个来源都有
	// default_max_age 策略，缺失即 fail-closed。
	RequireValidityPolicyForEachSource bool `yaml:"require_validity_policy_for_each_source" json:"require_validity_policy_for_each_source"`
	// DefaultMaxAge 来源 → 允许最大年龄（自 ObservedAt 起推导 ValidUntil）。
	DefaultMaxAge map[string]Duration `yaml:"default_max_age" json:"default_max_age"`
	// ExcludeFutureDated 丢弃 ValidFrom 晚于当前时刻的 Evidence。
	ExcludeFutureDated bool `yaml:"exclude_future_dated" json:"exclude_future_dated"`
	// FailClosedOnUnknownValidity 有效期未知且无来源策略时丢弃（不进答案/
	// 缓存）。必须显式配置为 true（指针区分"未配置"与 false；v1.2 无
	// 放开路径，配置 false 被校验拒绝——fail-closed 是安全不变量）。
	FailClosedOnUnknownValidity *bool `yaml:"fail_closed_on_unknown_validity" json:"fail_closed_on_unknown_validity"`
}

// EvidenceCacheConfig 是 Evidence 缓存配置（F-004；docs/design/04 §2）。
type EvidenceCacheConfig struct {
	Enabled             bool     `yaml:"enabled" json:"enabled"`
	Implementation      string   `yaml:"implementation" json:"implementation"`
	MaxEntries          int      `yaml:"max_entries" json:"max_entries"`
	MaxBytes            int64    `yaml:"max_bytes" json:"max_bytes"`
	MaxItemBytes        int64    `yaml:"max_item_bytes" json:"max_item_bytes"`
	TTL                 Duration `yaml:"ttl" json:"ttl"`
	CacheEmptyResults   bool     `yaml:"cache_empty_results" json:"cache_empty_results"`
	CacheProviderErrors bool     `yaml:"cache_provider_errors" json:"cache_provider_errors"`
	// Include 参与 cache key 的维度；须包含核心三项
	// normalized_question/config_version/principal_scope_fingerprint。
	Include []string `yaml:"include" json:"include"`
}

// DecisionConfig 是决策链配置（provider、Jev 参数与降级链）。
type DecisionConfig struct {
	Provider string    `yaml:"provider" json:"provider"`
	Jev      JevConfig `yaml:"jev" json:"jev"`
	// AttributeAllowlist 是 CandidateGroup.Attributes 的白名单；
	// 空表 = 剔除全部属性（最严格，防高基数/敏感渗漏）。
	AttributeAllowlist []string       `yaml:"attribute_allowlist,omitempty" json:"attribute_allowlist,omitempty"`
	Fallback           FallbackConfig `yaml:"fallback" json:"fallback"`
}

// JevConfig 是 Jev adapter 参数。密钥只含环境变量引用名，绝无明文。
type JevConfig struct {
	EndpointEnv  string `yaml:"endpoint_env" json:"endpoint_env"`
	APIKeyEnv    string `yaml:"api_key_env" json:"api_key_env"`
	Model        string `yaml:"model" json:"model"`
	ModelVersion string `yaml:"model_version" json:"model_version"`
	// MaxAttempts 最大尝试次数；默认/推荐 1（docs/design/03 §3：Jev 默认最多
	// 调用一次，以免重试耗尽补检/余量预算）。
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts"`
	// Timeout 单次调用超时（可选）；未配置时默认 Rejudge 预算，校验要求
	// ≤ Rejudge。
	Timeout *Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	// RetryBackoff 重试间隔（可选，默认 0）；最坏耗时 = Σ(attempt+backoff)
	// 必须装入 Rejudge 预算。
	RetryBackoff *Duration `yaml:"retry_backoff,omitempty" json:"retry_backoff,omitempty"`
	// BalanceEndpointEnv 余额查询端点的环境变量名（v1.3 F-016；O-2 协议
	// 未定按最小契约 + fake 先行，协议到位仅改 internal/balance 的 adapter）。
	BalanceEndpointEnv string `yaml:"balance_endpoint_env,omitempty" json:"balance_endpoint_env,omitempty"`
	// Breaker 熔断配置（v1.3 F-019；nil = 默认开启 {5 次连续失败, 30s 冷却}）。
	Breaker *JevBreakerConfig `yaml:"breaker,omitempty" json:"breaker,omitempty"`
}

// JevBreakerConfig 是 Jev 级熔断参数（Story 8.5.2.1）。
type JevBreakerConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// FailureThreshold 连续失败进 open 的阈值（必须 > 0）。
	FailureThreshold int `yaml:"failure_threshold" json:"failure_threshold"`
	// Cooldown open 冷却时长（必须 > 0；期满转 half_open 单探测）。
	Cooldown Duration `yaml:"cooldown" json:"cooldown"`
}

// FallbackConfig 是降级链配置；顺序固定 jev → deterministic_rules →
// decision_cache → abstain（docs/design/04 §3）。
type FallbackConfig struct {
	Order []string            `yaml:"order" json:"order"`
	Rules []RuleConfig        `yaml:"rules" json:"rules"`
	Cache DecisionCacheConfig `yaml:"cache" json:"cache"`
}

// RuleConfig 是单条确定性规则（条件完整、输出有限 action，可审计）。
type RuleConfig struct {
	Name   string `yaml:"name" json:"name"`
	When   string `yaml:"when" json:"when"`
	Action string `yaml:"action" json:"action"`
	// Threshold 条件参数（如置信度阈值，∈[0,1]）；
	// decision_confidence_below_threshold 条件必填。
	Threshold *float64 `yaml:"threshold,omitempty" json:"threshold,omitempty"`
}

// 内置规则条件名（docs/02 rules 样例；rules 引擎按注册表解析，
// 未知条件名在配置校验期拒绝）。
const (
	// CondValidEvidenceZero：全部候选组合计有效证据数为 0。
	CondValidEvidenceZero = "valid_evidence_count_equals_zero"
	// CondConfidenceBelowThreshold：上游决策置信度低于阈值（需 Threshold）。
	CondConfidenceBelowThreshold = "decision_confidence_below_threshold"
)

// KnownRuleConditions 是已知规则条件集（配置校验用）。
var KnownRuleConditions = map[string]bool{
	CondValidEvidenceZero:        true,
	CondConfidenceBelowThreshold: true,
}

// DecisionCacheConfig 是受约束决策缓存配置。
type DecisionCacheConfig struct {
	Enabled                                bool     `yaml:"enabled" json:"enabled"`
	TTL                                    Duration `yaml:"ttl" json:"ttl"`
	MaxEntries                             int      `yaml:"max_entries" json:"max_entries"`
	IncludeConfigVersion                   bool     `yaml:"include_config_version" json:"include_config_version"`
	IncludeScopeFingerprint                bool     `yaml:"include_scope_fingerprint" json:"include_scope_fingerprint"`
	IncludeCandidateFingerprint            bool     `yaml:"include_candidate_fingerprint" json:"include_candidate_fingerprint"`
	ValidateActionAgainstCurrentCandidates bool     `yaml:"validate_action_against_current_candidates" json:"validate_action_against_current_candidates"`
}

// AnswerConfig 是回答组装配置（引用安全，F-010）。
type AnswerConfig struct {
	MaxCitations                    int  `yaml:"max_citations" json:"max_citations"`
	RequireCitationForFactualClaims bool `yaml:"require_citation_for_factual_claims" json:"require_citation_for_factual_claims"`
	// AllowAnswerWithoutEvidence 默认 false：answer 必须至少有一条有效引用，
	// 除非业务批准无 Evidence 的纯对话模式。
	AllowAnswerWithoutEvidence bool   `yaml:"allow_answer_without_evidence" json:"allow_answer_without_evidence"`
	OutputOnUnknown            string `yaml:"output_on_unknown" json:"output_on_unknown"`
}

// HealthConfig 是 Health 聚合配置（F-005）。
type HealthConfig struct {
	RequiredComponents                      []string `yaml:"required_components" json:"required_components"`
	OptionalComponents                      []string `yaml:"optional_components" json:"optional_components"`
	CheckTimeout                            Duration `yaml:"check_timeout" json:"check_timeout"`
	DegradedWhenOptionalProviderUnavailable bool     `yaml:"degraded_when_optional_provider_unavailable" json:"degraded_when_optional_provider_unavailable"`
}

// MetricsConfig 是 Prometheus collector 配置（F-006；KAS 不开 listener）。
type MetricsConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	Namespace      string   `yaml:"namespace" json:"namespace"`
	LabelAllowlist []string `yaml:"label_allowlist" json:"label_allowlist"`
}

// TracingConfig 是 trace 配置（F-009）；正文捕获在 v1.2 强制关闭（fail-closed）。
type TracingConfig struct {
	Enabled             bool     `yaml:"enabled" json:"enabled"`
	CaptureQuestionText bool     `yaml:"capture_question_text" json:"capture_question_text"`
	CaptureEvidenceText bool     `yaml:"capture_evidence_text" json:"capture_evidence_text"`
	Include             []string `yaml:"include" json:"include"`
}
