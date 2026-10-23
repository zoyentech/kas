# Feature 6.1.1 — 组件健康与聚合

> 状态：done ｜ Epic 6.1 ｜ 粗估：1.5 人日

## 目标

HealthReport 类型 + 聚合器 + kas_health_status 指标数据面。

## 设计要点

- `ComponentHealth`（每组件状态与安全描述）；`HealthReport{Status, Components}`。
- required_components / optional_components 来自配置（docs/02）。
- 每组件独立 ctx（check_timeout），并发探测，聚合取最差。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 6.1.1.1 | Health 实现 | 3 |

## 验收标准

- 三态判定表驱动；degraded_when_optional_provider_unavailable 开关语义正确。

## 需求关联

F-005（P0）。
