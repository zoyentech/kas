# Epic 7.2 — 发布与文档

> 状态：todo ｜ Stage 7 ｜ 关联需求：docs/01 §7 首发门禁 ｜ 粗估：3 人日

## 目标与价值

发布前安全/合规审计与 v1.2.0 开源发布：敏感数据零泄漏证明、依赖漏洞/license 检查、README/godoc/CHANGELOG/MIGRATION 定稿、tag 与 Release、验收产物归档。

## 验收（Epic DoD）

- 审计报告通过（敏感数据扫描、govuln、license 兼容）。
- GitHub 上 v1.2.0 Release 完整：说明、产物链接、报告归档。
- GoldenSet 门槛达标记录（或明确未达标→不发布，按 docs/05 §4 处理）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 7.2.1 | 安全与合规审查 | 7.2.1.1 发布前审计 |
| 7.2.2 | v1.2.0 发布 | 7.2.2.1 文档与发布 |

## 依赖

- Epic 7.1 达标。

## 关联设计

docs/01 §7；docs/05 §4；docs/06（迁移指南素材）。
