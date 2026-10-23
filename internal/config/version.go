package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// VersionLength 是 config_version 的 hex 字符数（SHA-256 前 12 hex = 48 bit，
// 足以区分配置代次且对指标/trace 友好）。
const VersionLength = 12

// Version 生成脱敏 config_version：SHA-256(Canonical(cfg)) 前 12 hex
// （Story 2.2.1.1 T3）。
//
// 口径（docs/design/01 §6、docs/design/03 §5）：
//   - 幂等：同一配置恒得同一版本（与解析自何种字段顺序的 YAML 无关）
//   - 跨进程稳定：不依赖 map 遍历序、时间、环境
//   - secret 明文不参与（schema 无明文字段）；*_env 引用名参与——引用名
//     变化即配置变化，版本随之变化
//   - 消费方：S4 Evidence cache key 维度、S6 trace 字段、Reload 返回值；
//     同一 snapshot 内请求、trace、响应元数据和 cache key 使用同一版本
func Version(cfg Config) (string, error) {
	b, err := Canonical(cfg)
	if err != nil {
		return "", fmt.Errorf("config canonical 序列化失败: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:VersionLength], nil
}

// MustVersion 供测试与构造器使用；校验已通过的配置不会失败。
func MustVersion(cfg Config) string {
	v, err := Version(cfg)
	if err != nil {
		panic(err)
	}
	return v
}
