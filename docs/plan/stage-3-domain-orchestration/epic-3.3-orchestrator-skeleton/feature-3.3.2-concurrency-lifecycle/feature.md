# Feature 3.3.2 — 并发与生命周期

> 状态：done ｜ Epic 3.3 ｜ 粗估：1 人日

## 目标

F-012：Client 全方法并发安全；Close 停新请求、关闭 SDK 自有资源；宿主共享资源（HTTP client/registry）归宿主管理的边界写清。

## 设计要点

- 并发安全基线测试：Answer/Health/Reload/Close 混合并发 `-race`。
- Close 后调用 → ErrClosed；进行中请求不强制中断（由 ctx 控制）。
- 自有资源清单（进程内 cache、订阅者等）在 Close 中逐项关闭。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 3.3.2.1 | Client 生命周期与并发基线 | 2 |

## 验收标准

- docs/05 §2"并发 Answer/Health/Reload"行的日常化版本在 CI 常跑。
- 资源归属边界写入 godoc 与 CONTRIBUTING。

## 需求关联

F-012（P0）、F-011（并发面共同验证）。
