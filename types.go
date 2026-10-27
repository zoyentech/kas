package kas

import (
	"github.com/zoyentech/kas/internal/balance"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence"
	"github.com/zoyentech/kas/internal/evidence/perm"
)

// 本文件集中定义根包公共类型别名（D-7：实现全部在 internal/；类型唯一
// 权威定义在 internal/domain，别名保持公共 API 面稳定）。

// —— Request / Response（docs/design/03 §2）——

// Request 是一次问答请求。宿主先完成认证授权并构造可信 PrincipalScope；
// KAS 不从 question 推导 scope。
type Request = domain.Request

// PrincipalScope 由宿主可信认证/授权层生成；字段参与 Evidence cache key
// （F-004），不进入日志/指标。
type PrincipalScope = domain.PrincipalScope

// KnowledgeScope 标识知识空间与内容代次。
type KnowledgeScope = domain.KnowledgeScope

// ConversationContext 是受限会话上下文。
type ConversationContext = domain.ConversationContext

// Response 是问答结果；Degradation 为受控 reason code。
type Response = domain.Response

// ActionType 是有限决策动作枚举。
type ActionType = domain.ActionType

// 决策动作四枚举。
const (
	ActionAnswer     = domain.ActionAnswer
	ActionSupplement = domain.ActionSupplement
	ActionClarify    = domain.ActionClarify
	ActionAbstain    = domain.ActionAbstain
)

// Citation 指向最终有效 Evidence 的一条引用（F-010）。
type Citation = domain.Citation

// 降级 reason code 常量（Response.Degradation 取值，docs/design/04 §3）。
const (
	ReasonJevUnavailable      = domain.ReasonJevUnavailable
	ReasonJevContractError    = domain.ReasonJevContractError
	ReasonRulesApplied        = domain.ReasonRulesApplied
	ReasonDecisionCacheHit    = domain.ReasonDecisionCacheHit
	ReasonDecisionExhausted   = domain.ReasonDecisionExhausted
	ReasonDeadlineExceeded    = domain.ReasonDeadlineExceeded
	ReasonEvidenceUnavailable = domain.ReasonEvidenceUnavailable
	ReasonNoValidEvidence     = domain.ReasonNoValidEvidence
)

// ValidateRequest 前置校验（Question 非空、principal 三要素、RequestID 格式）。
func ValidateRequest(req Request) error { return domain.ValidateRequest(req) }

// —— Evidence 合约（docs/design/03 §4）——

type (
	// Evidence 是统一证据单元。
	Evidence = domain.Evidence
	// EvidenceRef 是候选组对 Evidence 的稳定引用。
	EvidenceRef = domain.EvidenceRef
	// EvidenceQuery 是检索请求参数。
	EvidenceQuery = domain.EvidenceQuery
	// EvidenceResult 是检索结果集。
	EvidenceResult = domain.EvidenceResult
	// EvidenceProvider 从宿主权威来源返回统一 Evidence；必须响应 ctx.Done()。
	EvidenceProvider = domain.EvidenceProvider
)

// —— DecisionProvider（docs/design/03 §3）——

type (
	// CandidateGroup 是证据候选组。
	CandidateGroup = domain.CandidateGroup
	// DecisionInput 是决策输入。
	DecisionInput = domain.DecisionInput
	// Decision 是决策输出。
	Decision = domain.Decision
	// DecisionProvider 将候选组映射为有限 action（F-007）。
	DecisionProvider = domain.DecisionProvider
	// ComponentHealth 是组件自报健康状态。
	ComponentHealth = domain.ComponentHealth
	// ComponentStatus 是组件健康枚举。
	ComponentStatus = domain.ComponentStatus
	// HealthReport 是 Health 聚合结果。
	HealthReport = domain.HealthReport
	// OverallStatus 是总健康状态。
	OverallStatus = domain.OverallStatus
	// HealthChecks 是宿主注入的额外检查点。
	HealthChecks = domain.HealthChecks
)

// 组件/总健康状态枚举。
const (
	ComponentHealthy  = domain.ComponentHealthy
	ComponentDegraded = domain.ComponentDegraded
	ComponentNotReady = domain.ComponentNotReady
	ComponentUnknown  = domain.ComponentUnknown

	StatusHealthy  = domain.StatusHealthy
	StatusDegraded = domain.StatusDegraded
	StatusNotReady = domain.StatusNotReady
)

// —— Cache / Trace 扩展点（docs/design/03 §6）——

type (
	// CacheKey 是 Evidence 缓存键维度（F-004）。
	CacheKey = domain.CacheKey
	// EvidenceCache 是缓存扩展点。
	EvidenceCache = domain.EvidenceCache
	// TraceEvent 是 trace 记录（白名单字段）。
	TraceEvent = domain.TraceEvent
	// TraceSink 由宿主注入。
	TraceSink = domain.TraceSink
)

// —— 余额监测（v1.3 F-016；v1.4 多源）——

type (
	// BalanceInfo 是一次余额查询结果（Amount/Unit 有界枚举）。
	BalanceInfo = balance.Info
	// BalanceState 是余额四态（normal/low/critical/unknown）。
	BalanceState = balance.State
	// BalanceSource 是余额来源契约（宿主可注入任意计费源）。
	BalanceSource = balance.Source
)

// BalanceConfig 是余额监测配置（全局默认 + balance.sources 按源覆盖）。
type BalanceConfig = config.BalanceConfig

// 余额四态枚举。
const (
	BalanceNormal   = balance.StateNormal
	BalanceLow      = balance.StateLow
	BalanceCritical = balance.StateCritical
	BalanceUnknown  = balance.StateUnknown
)

// —— 权限判定（O-7 双模式）——

// PermissionChecker 判定单条 Evidence 对当前 principal scope 是否可用；
// 未授权 Evidence 以 unauthorized 原因过滤（F-003/F-010）。
// 默认声明式实现见 internal/evidence/perm.TagChecker（scope fingerprint →
// 允许的 PermissionTags 集合）；宿主亦可注入判定函数（evidence.FuncChecker）。
type PermissionChecker = evidence.Checker

// NewTagChecker 由 scope fingerprint → 允许 tags 映射构建声明式权限判定器。
func NewTagChecker(allowed map[string][]string) PermissionChecker {
	return perm.NewTagChecker(allowed)
}
