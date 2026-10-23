package kas

import (
	"io"
	"time"
)

// TODO(S2): Story 2.1.1.1 落地完整配置类型系统；此处为 docs/design/03 §1 占位。

// TimeoutBudget 是五阶段请求预算；total 必须精确等于其余五段之和（F-001）。
type TimeoutBudget struct {
	Total       time.Duration
	Preliminary time.Duration
	FirstRound  time.Duration
	Rejudge     time.Duration
	Supplement  time.Duration
	Reserve     time.Duration
}

// Config 是 v1.2 配置模型（docs/design/02 示例 YAML 的 Go 形态）。
type Config struct {
	SchemaVersion int
	Timeouts      TimeoutBudget
	// Evidence、Decision、Answer、Health、Metrics、Tracing 等节在 Story 2.1.1.1 补齐。
}

// LoadConfig 从 r 解析 YAML（严格模式，未知字段拒绝）并返回 Config。
func LoadConfig(r io.Reader) (Config, error) {
	return loadConfig(r) // TODO(S2): Story 2.1.1.2
}

// LoadConfigFile 从 path 读取配置文件并解析。
func LoadConfigFile(path string) (Config, error) {
	return loadConfigFile(path) // TODO(S2): Story 2.1.1.2
}

// ValidateConfig 执行全部启动校验：预算等式、duration 溢出、provider 超时
// 装入阶段预算、reserve 下限、缓存上限与有效期策略（docs/design/04 §1）。
func ValidateConfig(cfg Config) error {
	return validateConfig(cfg) // TODO(S2): Feature 2.1.2
}
