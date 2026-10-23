# Feature 5.1.1 — 决策接口

> 状态：done ｜ Epic 5.1 ｜ 粗估：2 人日

## 目标

DecisionProvider 契约与响应校验器：任何实现（Jev/规则/缓存）返回的 Decision 都过同一校验器（F-010 第一道闸）。

## 设计要点

- 类型族：CandidateGroup（Attributes 仅白名单字段）、DecisionInput、Decision（含 Provider/Model/ModelVersion/Degraded）。
- 校验器：Action ∈ 枚举；TargetGroupIDs ⊆ 输入组 ID；Confidence 有限数值；ReasonCode 非自由文本直通。
- 校验失败 → ErrProviderContract + 事件（S6 指标）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 5.1.1.1 | 决策类型与校验器 | 3 |

## 验收标准

- docs/05 §2"Jev 非法 action/越界 ID/NaN → 拒绝并降级"的校验部分单测完备。

## 需求关联

F-007、F-010（P0）。
