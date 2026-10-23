package kas

// TODO(S3): Story 3.1.1.1 落地；此处为 docs/design/03 §2 占位。

// Request 是一次问答请求。宿主先完成认证授权并构造可信 PrincipalScope；
// KAS 不从 question 推导 scope。
type Request struct {
	Question     string
	Principal    PrincipalScope
	Knowledge    KnowledgeScope
	Conversation ConversationContext
	RequestID    string
}

// PrincipalScope 由宿主可信认证/授权层生成。原始用户 ID、token 不得进入
// 日志/指标/常规 trace；scope 字段参与 Evidence cache key（F-004）。
type PrincipalScope struct {
	TenantKey          string // 不透明内部标识，不进入日志/指标
	ScopeFingerprint   string // 宿主基于有效权限范围生成
	AuthorizationEpoch string // 权限变化时变更
}

// KnowledgeScope 标识知识空间与内容代次；corpus generation 变化后不复用
// 上一代缓存条目。
type KnowledgeScope struct {
	Namespace        string
	CorpusGeneration string
}

// ConversationContext 是受限的会话上下文（首版仅透传必要字段）。
type ConversationContext struct{} // TODO(S3): 按 Story 3.1.1.1 细化

// Response 是问答结果。answer 必须至少有一条有效引用，除非业务批准无
// Evidence 的纯对话模式（docs/design/02 allow_answer_without_evidence）。
type Response struct {
	Action        ActionType
	Answer        string
	Citations     []Citation
	Confidence    *float64
	TraceID       string
	ConfigVersion string
	Degraded      bool
	Degradation   []string // 受控 reason code，不含上游错误原文
}

// Citation 指向最终有效 Evidence 的一条引用（F-010：只能来自当前候选）。
type Citation struct {
	EvidenceID string // TODO(S3/S5): Story 5.4.1 细化字段（来源、定位、版本）
}

// ActionType 是有限决策动作枚举；未知值按 provider contract error 拒绝。
type ActionType string

const (
	ActionAnswer     ActionType = "answer"
	ActionSupplement ActionType = "supplement"
	ActionClarify    ActionType = "ask_clarification"
	ActionAbstain    ActionType = "abstain"
)
