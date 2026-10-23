package config

import (
	"fmt"
	"io"
	"os"
)

// LoadConfig 从 r 解析 YAML（严格模式）并执行全部启动校验（Story 2.1.1.2 T1）。
// 返回错误满足 errors.Is(err, ErrInvalidConfig)；文本含字段路径，不含 secret
// 与整份配置回显。Reload 复用本入口。
func LoadConfig(r io.Reader) (Config, error) {
	if r == nil {
		return Config{}, singleError("(root)", "reader 为 nil")
	}
	var cfg Config
	if err := decodeStrict(r, &cfg); err != nil {
		return Config{}, err
	}
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadConfigFile 从 path 加载配置文件。
func LoadConfigFile(path string) (Config, error) {
	f, err := os.Open(path) // #nosec G304 -- 调用方指定路径属预期行为
	if err != nil {
		return Config{}, fmt.Errorf("%w: 打开配置文件失败: %v", ErrInvalidConfig, err)
	}
	defer func() { _ = f.Close() }()
	return LoadConfig(f)
}
