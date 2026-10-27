
# Epic 8.7 — 监控栈交付

> 状态：done ｜ Stage 8 ｜ 关联需求：F-015/F-017/F-018（P0，路线 A 承载）｜ 粗估：2 人日

## 目标与价值

"数据已导出"到"打开就能看"之间缺的是栈与看板：docker compose 一键起
Prometheus+Grafana（版本固定）、datasource/看板自动 provisioning、宿主
`/metrics` 可运行示例、监控指南与 PromQL 手册。

## 验收（Epic DoD）

- `cd monitoring && docker compose up -d` 后 Grafana :3000 打开即见 kas-overview 看板。
- 看板 JSON 结构校验（面板齐全/表达式非空/数据源引用正确）进入 CI。
- examples/metrics 运行后被栈抓取，看板出数（冒烟留档）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.7.1 | 栈、看板与示例 | 8.7.1.1 compose-stack；8.7.1.2 grafana-dashboard；8.7.1.3 host-example-guide |

## 依赖

- Epic 8.2（余额 gauge）、8.5.2（熔断 gauge）——看板面板消费其指标。

## 关联设计

ADR-0003（路线 A 决策）；docs/design/04 §5（指标语义）。
