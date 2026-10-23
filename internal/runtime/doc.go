// Package runtime 实现 Runtime Manager：不可变 Runtime Snapshot 构建、
// 脱敏 config_version 固化与 current pointer 的原子替换（docs/design/01 §6
// 发布顺序）。请求开始读取一次 snapshot 并全程持有，保证同一请求内版本一致。
package runtime
