# Feature 7.2.1 — 安全与合规审查

> 状态：todo ｜ Epic 7.2 ｜ 粗估：1 人日

## 目标

发布前最后一道审计：敏感数据零泄漏证明与依赖合规。

## 设计要点

- 敏感面：指标 label、trace 输出、错误信息、日志、panic 信息（docs/01 §7"敏感数据不出现在指标和常规日志"）。
- 依赖：`govulncheck`、license 兼容清单（Apache-2.0 生态）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 7.2.1.1 | 发布前审计 | 2 |

## 验收标准

- 审计报告归档 `docs/release-notes/v1.2.0-audit.md`。

## 需求关联

首发门禁（安全项）。
