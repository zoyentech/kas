# Feature 2.2.2 — 原子 Reload 与 snapshot

> 状态：todo ｜ Epic 2.2 ｜ 粗估：2.5 人日

## 目标

不可变 Runtime Snapshot 与原子替换：New 构建、Reload 五步流程（读取/解析 → Validate → 准备 provider/runtime → 生成版本 → 原子替换），失败保持 current pointer 原值（docs/01 §6）。

## 设计要点

- snapshot 内持有：已验证 Config、config_version、预算、缓存参数、provider 引用（由 Dependencies 注入或 Reload 重建）。
- 替换用 `atomic.Pointer[runtimeSnapshot]`；请求开始读一次并全程持有（docs/01 §3 编排器条目）。
- New 同步校验：预算等式、必要依赖、cache 容量、有效期策略、provider 参数（docs/03 §1）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 2.2.2.1 | Runtime Snapshot 构建 | 2 |
| 2.2.2.2 | 原子替换语义 | 3 |

## 验收标准

- 并发"Answer 进行中多次 Reload"：在途请求 Response.ConfigVersion 恒为开始时版本；新请求立即新版本。
- 无效 Reload 返回错误且版本/行为不变。

## 需求关联

F-011（P0）、F-012（无状态前提）。
