# Feature 6.2.1 — collector 与宿主接入

> 状态：todo ｜ Epic 6.2 ｜ 粗估：3.5 人日

## 目标

十类指标实现 + 事件消费 + Registerer 注入 + label 防护。

## 设计要点

- 指标清单（docs/04 §5）：requests_total、request_duration_seconds、stage_duration_seconds、stage_deadline_exceeded_total、deadline_exceeded_total、evidence_cache_total、evidence_cache_evictions_total、evidence_filtered_total、provider_errors_total、decision_fallback_total、health_status。
- label_allowlist（docs/02）在 collector 内强制：不在名单的 label 值被拒（错误可见）。
- namespace 前缀（kas）与宿主 registry 注入（Option）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 6.2.1.1 | 指标族实现 | 3 |
| 6.2.1.2 | 宿主接入与增量断言 | 2 |

## 验收标准

- 全指标族可被 promtest 工具抓取并断言值/标签。

## 需求关联

F-006（P0）。
