# Epic 5.1 — 决策契约与规则引擎

> 状态：done ｜ Stage 5 ｜ 关联需求：F-007/F-010（P0）｜ 粗估：3.5 人日

## 目标与价值

DecisionProvider 契约定型（编排器与决策协议解耦，F-007）+ 响应安全校验（有限 action、目标组归属、数值合法）+ 确定性规则引擎（可审计、条件完整、输出有限 action），规则同时服务初判 hook 与降级链。

## 验收（Epic DoD）

- docs/03 §3 全部类型落地；非法 Decision（未知 action/越界 ID/NaN）被统一校验器拒绝。
- 规则引擎可求值 docs/02 rules 样例并输出可解释 reason/name。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 5.1.1 | 决策接口 | 5.1.1.1 决策类型与校验 |
| 5.1.2 | 规则引擎 | 5.1.2.1 规则求值 |

## 依赖

- S3（仅 Epic 3.1，可与 S4 并行）。

## 关联设计

docs/03 §3（DecisionProvider 全节）；docs/02 decision.fallback.rules；docs/04 §3（规则可审计要求）。
