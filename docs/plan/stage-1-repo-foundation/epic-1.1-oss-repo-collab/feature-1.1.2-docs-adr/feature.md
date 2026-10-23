# Feature 1.1.2 — 文档结构与决策记录

> 状态：done ｜ Epic 1.1 ｜ 粗估：1 人日

## 目标

把设计文件包与本计划迁入仓库，建立 ADR（Architecture Decision Record）机制，使设计与实现的每处偏差可追溯。

## 设计要点

- 目录：`docs/design/`（kas-v1.2 设计包 01~06 原样迁入）、`docs/plan/`（本 WBS）、`docs/adr/`（决策记录）。
- ADR 流程：提案 → 编号 → 状态（proposed/accepted/superseded）；实现与设计冲突必须先 ADR。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 1.1.2.1 | 文档目录与 ADR 机制 | 4 |

## 验收标准

- 文档树与设计包内容一致（迁移不改动正文）。
- ADR-0001/0002 归档且被 overview/01 引用。

## 需求关联

工程基线（无 F-xxx）。
