package kas

import (
	"io"

	"github.com/kas-runtime/kas/internal/config"
)

// Config 是 v1.2 配置模型（internal/config 为唯一权威定义；本别名为公共
// API 形态，字段与 docs/design/02 示例 YAML 逐节对应）。配置只含 *_env
// 引用名，绝无 secret 明文。
type Config = config.Config

// TimeoutBudget 是五阶段请求预算；Total 必须精确等于其余五段之和（F-001）。
type TimeoutBudget = config.TimeoutBudget

// Duration 是配置中的时长类型（YAML 按 Go time.Duration 字符串格式）。
type Duration = config.Duration

// LoadConfig 从 r 解析 YAML（严格模式，未知字段拒绝）并执行全部启动校验。
// 错误满足 errors.Is(err, ErrInvalidConfig)，文本含字段路径、不含 secret
// 与整份配置回显。
func LoadConfig(r io.Reader) (Config, error) { return config.LoadConfig(r) }

// LoadConfigFile 从 path 读取配置文件并解析校验。
func LoadConfigFile(path string) (Config, error) { return config.LoadConfigFile(path) }

// ValidateConfig 执行全部启动校验：预算等式（F-001）、duration 溢出、
// provider 超时装入阶段预算、reserve 下限、缓存上限与有效期策略
// （docs/design/04 §1）。多条违规聚合逐条列出。
func ValidateConfig(cfg Config) error { return config.Validate(cfg) }

// ConfigVersion 生成脱敏 config_version（SHA-256 前 12 hex）。幂等且跨进程
// 稳定；secret 明文不参与，引用名参与（docs/design/01 §6）。
func ConfigVersion(cfg Config) (string, error) { return config.Version(cfg) }
