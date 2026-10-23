package kas

import (
	"context"
	"time"
)

// TODO(S4): Epic 4.3 落地；此处为 docs/design/03 §6 占位。

// CacheKey 是 Evidence 缓存键维度：规范化问题、config_version、principal
// scope fingerprint、authorization epoch、知识空间、内容代次与检索策略版本
// （F-004；docs/design/04 §2）。存储仅使用不可逆 key 摘要。
type CacheKey struct{} // TODO(S4): Story 4.3.1.2 细化

// EvidenceCache 是缓存扩展点（进程内默认实现；共享缓存是 post-MVP adapter）。
// 读 cache 后仍用当前权限与当前时刻复核；仅缓存 Evidence，不缓存最终回答。
type EvidenceCache interface {
	Get(ctx context.Context, key CacheKey) (EvidenceResult, bool, error)
	Put(ctx context.Context, key CacheKey, value EvidenceResult, ttl time.Duration) error
	DeleteNamespace(ctx context.Context, namespace string) error
}
