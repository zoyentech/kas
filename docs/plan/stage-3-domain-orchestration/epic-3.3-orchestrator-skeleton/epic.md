# Epic 3.3 — 编排器骨架

> 状态：done ｜ Stage 3 ｜ 关联需求：F-002/F-007/F-012（P0）｜ 粗估：4 人日

## 目标与价值

实现 `Client.Answer` 主流程骨架：读快照→固定版本→五阶段按预算执行→组装 Response 的空转版本；内部事件总线为 S6（metrics/trace）提供唯一数据面；Client 并发安全与 Close 语义（F-012）。S4/S5 以挂载点方式接入，不需要改主流程结构。

## 验收（Epic DoD）

- fake provider 注入下骨架端到端测试通过：五阶段按序执行、阶段预算被消费、事件按序发出、Response.ConfigVersion 恒定。
- F-012：Answer/Health/Reload 并发 `-race` 通过；Close 后 ErrClosed。
- 事件字段覆盖 docs/04 §5 全部指标与 §6 全部 trace 字段的需求（S6 只做消费者）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 3.3.1 | 阶段流水线 | 3.3.1.1 Answer 主流程；3.3.1.2 事件总线 |
| 3.3.2 | 并发与生命周期 | 3.3.2.1 Client 生命周期 |

## 依赖

- Epic 3.1/3.2；S2 snapshot。

## 关联设计

docs/01 §3（编排器职责）、§4（端到端流程 10 步）；docs/04 §5/§6（事件需求来源）。
