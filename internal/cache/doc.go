// Package cache 实现问题规范化（NFKC/trim/空白折叠）、安全 cache key
// （含 scope/epoch/版本维度、仅存不可逆摘要）与有界内存 Evidence 缓存。
// 设计依据：docs/design/04 §2（Evidence 缓存）。
package cache
