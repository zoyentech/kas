package domain

import (
	"context"
	"time"
)

// CacheKey 是 Evidence 缓存键的全部维度（F-004；docs/design/04 §2）：
// 规范化问题、config_version、principal scope fingerprint、authorization
// epoch、知识空间、内容代次与检索策略版本。存储仅使用不可逆 key 摘要，
// 不保留原始问题和身份明文。哪些维度实际参与由配置 evidence.cache.include
// 决定（核心三项强制）。
type CacheKey struct {
	QuestionDigest         string // 规范化问题的不可逆摘要（hex）
	ConfigVersion          string
	ScopeFingerprint       string // 不可逆摘要形态由 cache 层决定
	AuthorizationEpoch     string
	KnowledgeNamespace     string
	CorpusGeneration       string
	RetrievalPolicyVersion string
}

// EvidenceCache 是缓存扩展点（进程内默认实现；共享缓存是 post-MVP
// adapter，须保持相同 scope/content generation/config version/TTL 与
// 权限复核规则）。读 cache 后仍用当前权限与当前时刻复核；仅缓存
// Evidence，不缓存最终回答。
type EvidenceCache interface {
	Get(ctx context.Context, key CacheKey) (EvidenceResult, bool, error)
	Put(ctx context.Context, key CacheKey, value EvidenceResult, ttl time.Duration) error
	DeleteNamespace(ctx context.Context, namespace string) error
}

// TraceEvent 是一次请求内的 trace 记录。字段受 include 白名单约束；
// question 正文、Evidence 正文、身份明文、secret 禁止进入（S6 细化）。
type TraceEvent struct {
	TraceID       string            `json:"trace_id"`
	ConfigVersion string            `json:"config_version"`
	Fields        map[string]string `json:"fields"` // 白名单字段；值须脱敏
}

// TraceSink 由宿主注入；KAS 不落盘、不外发 trace。
type TraceSink interface {
	Record(ctx context.Context, event TraceEvent)
}
