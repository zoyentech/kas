// Package config 实现配置类型系统、YAML 严格解析、启动校验（预算等式、
// provider 策略）、脱敏 config_version 与运行时快照的原子 Reload。
// 设计依据：docs/design/02（示例配置）、docs/design/04 §1（启动校验）。
package config
