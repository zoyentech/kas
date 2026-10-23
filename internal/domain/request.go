package domain

import (
	"errors"
	"strings"
)

// Request 是一次问答请求。宿主先完成认证授权并构造可信 PrincipalScope；
// KAS 不从 question 推导 scope（docs/design/03 §2）。
type Request struct {
	Question     string
	Principal    PrincipalScope
	Knowledge    KnowledgeScope
	Conversation ConversationContext
	RequestID    string
}

// PrincipalScope 由宿主可信认证/授权层生成。原始用户 ID、token 不得进入
// 日志/指标/常规 trace；ScopeFingerprint 与 AuthorizationEpoch 参与 Evidence
// cache key（F-004），权限变化时必须变更。
type PrincipalScope struct {
	TenantKey          string // 不透明内部标识，不进入日志/指标
	ScopeFingerprint   string // 宿主基于有效权限范围生成
	AuthorizationEpoch string // 权限变化时变更
}

// KnowledgeScope 标识知识空间与内容代次；CorpusGeneration 变化后不复用
// 上一代缓存条目。
type KnowledgeScope struct {
	Namespace        string
	CorpusGeneration string
}

// ConversationContext 是受限的会话上下文（v1.2 仅透传必要字段，不含历史
// 正文，防敏感扩散）。
type ConversationContext struct {
	// TurnID 当前轮次标识（由宿主会话系统生成，可空）。
	TurnID string
}

// ErrInvalidRequestDetail 携带字段级原因（不进公共错误串，仅日志/trace）。
type ErrInvalidRequestDetail struct {
	Field  string
	Reason string
}

// ValidateRequest 前置校验（Story 3.1.1.1 T1）：Question 非空、principal
// scope 三要素非空、knowledge namespace 非空、RequestID 可选但格式受限。
// 返回的 error 满足 errors.Is(err, ErrInvalidRequest)，不含问题正文。
func ValidateRequest(req Request) error {
	var fields []ErrInvalidRequestDetail
	if strings.TrimSpace(req.Question) == "" {
		fields = append(fields, ErrInvalidRequestDetail{Field: "Question", Reason: "不能为空"})
	}
	if req.Principal.TenantKey == "" {
		fields = append(fields, ErrInvalidRequestDetail{Field: "Principal.TenantKey", Reason: "不能为空"})
	}
	if req.Principal.ScopeFingerprint == "" {
		fields = append(fields, ErrInvalidRequestDetail{Field: "Principal.ScopeFingerprint", Reason: "不能为空"})
	}
	if req.Principal.AuthorizationEpoch == "" {
		fields = append(fields, ErrInvalidRequestDetail{Field: "Principal.AuthorizationEpoch", Reason: "不能为空"})
	}
	if req.Knowledge.Namespace == "" {
		fields = append(fields, ErrInvalidRequestDetail{Field: "Knowledge.Namespace", Reason: "不能为空"})
	}
	if req.RequestID != "" && !validRequestID(req.RequestID) {
		fields = append(fields, ErrInvalidRequestDetail{Field: "RequestID", Reason: "仅允许 [A-Za-z0-9._:-]，长度 1~64"})
	}
	if len(fields) == 0 {
		return nil
	}
	first := fields[0]
	return &InvalidRequestError{Fields: fields, wrap: errors.New(first.Field + ": " + first.Reason)}
}

// validRequestID 校验 RequestID 字符集与长度（防注入日志/索引系统）。
func validRequestID(id string) bool {
	if len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == ':' || r == '-':
		default:
			return false
		}
	}
	return len(id) > 0
}

// InvalidRequestError 聚合字段级校验失败原因。
type InvalidRequestError struct {
	Fields []ErrInvalidRequestDetail
	wrap   error
}

// Error 返回固定格式（仅字段名与原因，不含问题正文）。
func (e *InvalidRequestError) Error() string { return "kas: invalid request: " + e.wrap.Error() }

// Unwrap 使 errors.Is(err, ErrInvalidRequest) 成立。
func (e *InvalidRequestError) Unwrap() error { return ErrInvalidRequest }
