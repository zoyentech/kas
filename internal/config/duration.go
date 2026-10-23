package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration 是 YAML/JSON 配置中的时长类型：按 Go time.Duration 字符串格式
// 校验（如 4800ms、1.5h、720h、120s），底层单位为纳秒。YAML 中仅接受字符串
// 形态（docs/design/02：duration 按 Go time.Duration 格式校验）。
type Duration time.Duration

// D 便捷构造（供测试与程序化构造 Config 使用）。
func D(d time.Duration) Duration { return Duration(d) }

// Std 返回标准库 time.Duration。
func (d Duration) Std() time.Duration { return time.Duration(d) }

// String 返回 time.Duration 字符串形态。
func (d Duration) String() string { return time.Duration(d).String() }

// UnmarshalYAML 严格解析 duration 字符串；非法值返回含原值的错误
// （字段路径由 decode 层包装补充）。仅接受字符串标量（拒绝裸数字），
// 避免纳秒整数被静默当作 duration。
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Tag == "!!null" {
		*d = 0
		return nil
	}
	if node.Tag != "!!str" {
		return fmt.Errorf("duration 必须是字符串（如 4800ms、1.5h、720h），实际为 %v", node.Value)
	}
	v, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("duration 值 %q 不是合法的 Go time.Duration 格式（示例: 4800ms、1.5h、720h）", node.Value)
	}
	*d = Duration(v)
	return nil
}

// MarshalYAML 输出 time.Duration 字符串形态（规范化序列化使用 JSON 形态）。
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// MarshalJSON 以整数纳秒输出（canonical 序列化的稳定形态；与 YAML 字符串
// 形态不同属有意为之：JSON 仅供机器比对，YAML 供人阅读）。
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", int64(d))), nil
}
