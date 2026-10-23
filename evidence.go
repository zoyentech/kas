package kas

import (
	"context"
	"time"
)

// TODO(S4): Epic 4.1 落地；此处为 docs/design/03 §4 占位。

// Evidence 是统一证据单元。有效期优先使用来源显式时间（ValidFrom/ValidUntil）；
// 缺失时只能按批准的 source max-age 推导，策略缺失则 fail-closed。
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

// EvidenceRef 是候选组对 Evidence 的稳定引用。
type EvidenceRef struct {
	EvidenceID string // TODO(S4): Story 4.1.1.1 细化
}

// EvidenceQuery 是检索请求参数。
type EvidenceQuery struct{} // TODO(S4): Story 4.1.1.1 细化（规范化问题、top_k、namespace 等）

// EvidenceResult 是检索结果集。
type EvidenceResult struct{} // TODO(S4): Story 4.1.1.1 细化（Evidence 列表与来源元数据）

// EvidenceProvider 从宿主权威来源或检索服务返回统一 Evidence，不改变
// 事实源所有权。
type EvidenceProvider interface {
	Search(ctx context.Context, in EvidenceQuery) (EvidenceResult, error)
	Health(ctx context.Context) ComponentHealth
}
