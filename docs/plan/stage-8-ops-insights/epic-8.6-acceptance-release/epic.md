# Epic 8.6 — 验收与发布（v1.3.0）

> 状态：todo ｜ Stage 8 ｜ 关联需求：F-015~F-019 收口 ｜ 粗估：2 人日

## 目标与价值

ops 能力的验收套件（统计数学、诊断契约、仪表盘冒烟、稳定性行为）与 v1.3.0 发版
（复用既有发版流水线）。

## 验收（Epic DoD）

- 新增验收全绿并进 CI；GoldenSet 不回归。
- CHANGELOG/README/指南更新；v1.3.0 tag 走流水线全绿发布。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.6.1 | ops 验收套件 | 8.6.1.1 stats-acceptance；8.6.1.2 diag-dashboard-acceptance |
| 8.6.2 | v1.3.0 发布 | 8.6.2.1 v1.3.0-release |

## 依赖

- Epic 8.1~8.5 全部。

## 关联设计

stage-7 验收与发版模式（复用 runner/流水线约定）。
