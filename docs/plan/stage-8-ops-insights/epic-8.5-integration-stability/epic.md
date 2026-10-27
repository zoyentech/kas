# Epic 8.5 — 接入稳定化

> 状态：done ｜ Stage 8 ｜ 关联需求：F-019（P0）｜ 粗估：3 人日

## 目标与价值

v1.2.0 审核发现两处真实稳定性风险（panic 击穿、Jev 慢失败），并补齐宿主接入的
契约覆盖与指南。**本 Epic 优先级最高（8.5.1 先行）。**

## 验收（Epic DoD）

- panic provider 不击穿进程；事件 error_class=panic；契约套件新增 panic 用例。
- Jev 熔断三态迁移正确；open 期间零 Jev 调用；恢复半开自动回闭。
- 接入指南（含熔断/panic 语义）合入 docs；examples 增诊断面示例。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.5.1 | panic 隔离 | 8.5.1.1 panic-guard |
| 8.5.2 | Jev 熔断 | 8.5.2.1 breaker |
| 8.5.3 | 稳定性配套 | 8.5.3.1 contracts-guide-examples |

## 依赖

- S3 deadline 引擎；S5 降级链；S6 事件。

## 关联设计

docs/design/04 §1（provider ctx 合规——panic 隔离是其补全）；ADR-0003。
