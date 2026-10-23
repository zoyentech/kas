# Feature 5.3.2 — 受约束决策缓存

> 状态：todo ｜ Epic 5.3 ｜ 粗估：1.5 人日

## 目标

决策缓存的"受约束"实现：key 绑定 config/model/policy 版本+候选 fingerprint+知识代次+principal scope；命中后对当前候选重新校验 action 与目标组（docs/04 §3）。

## 设计要点

- 配置：enabled/ttl/max_entries/include_*（docs/02 decision.fallback.cache）。
- 命中再用 = 再过 ValidateDecision（当前输入组），不匹配视为 miss。
- 独立于 Evidence 缓存的小容量 LRU。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 5.3.2.1 | 决策缓存实现 | 3 |

## 验收标准

- docs/05 §2"决策缓存命中 → 候选/版本/scope 匹配且 action 重新校验"自动化。

## 需求关联

F-008（第三级）。
