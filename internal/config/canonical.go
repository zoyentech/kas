package config

import (
	"encoding/json"
	"sort"
)

// Canonical 返回配置的规范化字节串（Story 2.2.1.1 T1）：
//   - JSON 形态：struct 字段序稳定；map 键由 encoding/json 排序（稳定）
//   - duration 序列化为整数纳秒（Duration.MarshalJSON）
//   - 无序集合（label_allowlist / tracing.include / evidence.cache.include）
//     排序后参与，使"同一集合不同顺序"得到相同字节串；有序序列
//     （fallback.order、rules 优先级）保持原序
//   - 同一份逻辑配置以不同字段顺序书写时，解析后规范化结果逐字节相等
//   - 不修改入参（快照不可变性）：集合切片先复制再排序
//
// 该字节串是 config_version 的唯一输入（docs/design/01 §6：config_version
// 基于规范化、可观测配置生成；secret 明文不参与计算，secret 引用名/revision
// 参与——schema 中仅存在 *_env 引用名字段，无明文字段）。
func Canonical(cfg Config) ([]byte, error) {
	c := cfg // 浅拷贝；切片字段需深拷贝后再排序
	c.Metrics.LabelAllowlist = sortedCopy(cfg.Metrics.LabelAllowlist)
	c.Tracing.Include = sortedCopy(cfg.Tracing.Include)
	c.Evidence.Cache.Include = sortedCopy(cfg.Evidence.Cache.Include)
	return json.Marshal(c)
}

func sortedCopy(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}
