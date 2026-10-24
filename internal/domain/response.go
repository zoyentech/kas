package domain

// Response 是问答结果（docs/design/03 §2）。answer 必须至少有一条有效引用，
// 除非业务批准无 Evidence 的纯对话模式（config answer.allow_answer_without_evidence）。
type Response struct {
	Action        ActionType
	Answer        string
	Citations     []Citation
	Confidence    *float64 // 可空；出现 NaN/Inf 属契约违规
	TraceID       string
	ConfigVersion string
	Degraded      bool
	Degradation   []string // 受控 reason code（ReasonXxx 常量），非自由文本
}

// ActionType 是有限决策动作枚举；未知值按 provider contract error 拒绝。
type ActionType string

// 决策动作四枚举（docs/design/03 §2）。
const (
	ActionAnswer     ActionType = "answer"
	ActionSupplement ActionType = "supplement"
	ActionClarify    ActionType = "ask_clarification"
	ActionAbstain    ActionType = "abstain"
)

// ValidAction 判定 action 属于枚举。
func ValidAction(a ActionType) bool {
	switch a {
	case ActionAnswer, ActionSupplement, ActionClarify, ActionAbstain:
		return true
	}
	return false
}

// ParseAction 解析并校验 action 字符串；非法值返回 false。
func ParseAction(s string) (ActionType, bool) {
	a := ActionType(s)
	return a, ValidAction(a)
}

// Degradation reason code 常量表（docs/design/04 §3 降级链 + 错误映射）。
// 集中一处定义；新增 reason code 须评审（进入 Response.Degradation 与
// kas_decision_fallback_total 指标 label value）。
const (
	// ReasonJevUnavailable Jev 不可用（timeout/5xx/连接失败），已降级。
	ReasonJevUnavailable = "jev_unavailable"
	// ReasonJevContractError Jev 响应违反契约（未知 action/越界 ID/NaN）。
	ReasonJevContractError = "jev_contract_error"
	// ReasonRulesApplied 决策由确定性规则给出。
	ReasonRulesApplied = "rules_applied"
	// ReasonDecisionCacheHit 决策来自受约束决策缓存（已重新校验）。
	ReasonDecisionCacheHit = "decision_cache_hit"
	// ReasonDecisionExhausted 降级链耗尽，最终 abstain/clarify。
	ReasonDecisionExhausted = "decision_exhausted"
	// ReasonDeadlineExceeded 阶段或 root deadline 超时导致降级。
	ReasonDeadlineExceeded = "deadline_exceeded"
	// ReasonEvidenceUnavailable EvidenceProvider 不可用。
	ReasonEvidenceUnavailable = "evidence_unavailable"
	// ReasonNoValidEvidence 全部 Evidence 被过滤（过期/未生效/无权限/未知）。
	ReasonNoValidEvidence = "no_valid_evidence"
)

// Citation 指向最终有效 Evidence 的一条引用（F-010：只能来自当前候选；
// docs/design/03 §2"回答只引用最终有效 Evidence"）。
type Citation struct {
	EvidenceID string
	SourceID   string
	// Segment 可选锚点（如文档内段落/切片 ID）。
	Segment string `json:",omitempty"`
	// ContentVersion 引用时的知识内容版本（用于审计回放）。
	ContentVersion string `json:",omitempty"`
}
