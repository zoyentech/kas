package trace

import (
	"strings"

	"github.com/zoyentech/kas/internal/domain"
)

// sensitiveKeys 是敏感键硬拒清单（docs/design/01 §6：问题正文、Evidence
// 正文、身份明文、secret、prompt 不得进入 trace）。
var sensitiveKeys = []string{
	"question", "prompt", "evidence", "tenant", "principal", "user", "token", "secret", "api_key",
}

// Redactor 按 include 白名单过滤字段并硬拒敏感键（Story 6.3.1.1 T2）：
//   - include 为空 = 全部白名单默认字段
//   - 不在 include 中的字段丢弃
//   - 敏感键（含子串匹配）无条件丢弃，即使被误加入 include
type Redactor struct {
	Include map[string]bool
}

// NewRedactor 由 include 白名单构建；空列表使用默认白名单（docs/02）。
func NewRedactor(include []string) *Redactor {
	if len(include) == 0 {
		include = []string{
			"config_version", "evidence_provider", "decision_provider",
			"model", "model_version", "fallback_route", "cache_result",
			"stage", "stage_outcome", "provider", "duration_ms", "remaining_ms",
			"candidate_count", "filtered_count", "action", "reason_code", "outcome",
		}
	}
	m := map[string]bool{}
	for _, k := range include {
		m[k] = true
	}
	return &Redactor{Include: m}
}

// Filter 过滤 TraceEvent 字段：返回脱敏后事件（原事件不修改）。
func (r *Redactor) Filter(ev domain.TraceEvent) domain.TraceEvent {
	out := domain.TraceEvent{TraceID: ev.TraceID, ConfigVersion: ev.ConfigVersion, Fields: map[string]string{}}
	for k, v := range ev.Fields {
		if !r.Include[k] || isSensitive(k) || isSensitiveValue(v) {
			continue
		}
		out.Fields[k] = v
	}
	return out
}

func isSensitive(key string) bool {
	lk := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

// isSensitiveValue 防御：值中出现典型敏感形态（超长自由文本）时丢弃。
func isSensitiveValue(v string) bool {
	return len(v) > 512
}
