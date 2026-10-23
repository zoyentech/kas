package domain

import (
	"context"
	"time"
)

// Evidence 是统一证据单元（docs/design/03 §4）。有效期优先使用来源显式
// 时间（ValidFrom/ValidUntil）；缺失时只能按批准的 source max-age 推导，
// 策略缺失则 fail-closed（S4 过滤器实现）。
type Evidence struct {
	ID             string
	SourceID       string
	Title          string
	Excerpt        string
	ValidFrom      *time.Time
	ValidUntil     *time.Time
	ObservedAt     time.Time
	ContentVersion string
	PermissionTags []string
	Score          float64
}

// EvidenceRef 是候选组对 Evidence 的稳定引用（EvidenceID + 来源定位）。
type EvidenceRef struct {
	EvidenceID     string
	SourceID       string
	ContentVersion string
}

// EvidenceQuery 是检索请求参数。Question 已由调用方规范化（S4 cache 层）；
// TopK/Namespace 来自快照配置；ValidAt 用于有效期判定。
type EvidenceQuery struct {
	Question         string
	Namespace        string
	CorpusGeneration string
	TopK             int
	ValidAt          time.Time
}

// EvidenceResult 是检索结果集：Evidence 列表 + 来源/授权元数据快照。
// 空结果与错误不同：空结果可能可缓存（按 cache_empty_results 配置），
// provider error 一律不缓存。
type EvidenceResult struct {
	Evidence []Evidence
	// FromCache 标记本次结果来自缓存命中（仍需权限与有效期复核）。
	FromCache bool
}

// EvidenceProvider 从宿主权威来源或检索服务返回统一 Evidence，不改变
// 事实源所有权。实现必须响应 ctx.Done()（docs/design/04 §1）。
type EvidenceProvider interface {
	Search(ctx context.Context, in EvidenceQuery) (EvidenceResult, error)
	Health(ctx context.Context) ComponentHealth
}
