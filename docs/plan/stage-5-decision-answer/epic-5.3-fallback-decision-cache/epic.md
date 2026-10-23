# Epic 5.3 — 降级链与决策缓存

> 状态：todo ｜ Stage 5 ｜ 关联需求：F-008（P0）｜ 粗估：3.5 人日

## 目标与价值

实现固定顺序降级编排：Jev 一次尝试 → 确定性规则 → 受约束决策缓存 → abstain/clarify（docs/04 §3）。任何降级设置 degraded=true、输出受控 reason code、写 fallback route 指标与 trace；上游错误/思维链/prompt 不外泄。决策缓存命中后对当前候选再校验。

## 验收（Epic DoD）

- docs/05 §2 五行降级矩阵（Jev timeout/5xx、非法响应、规则冲突、决策缓存命中）全部自动化。
- 降级全程不超 root deadline；每级耗时计入阶段预算。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 5.3.1 | 降级编排 | 5.3.1.1 降级链执行器 |
| 5.3.2 | 受约束决策缓存 | 5.3.2.1 决策缓存实现 |

## 依赖

- Epic 5.1（规则）、5.2（Jev）；S3（事件总线、deadline）。

## 关联设计

docs/04 §3（降级顺序与决策缓存约束）；docs/02 decision.fallback 节；docs/05 §2。
