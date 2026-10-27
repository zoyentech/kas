# Epic 8.1 — 运行统计内核

> 状态：todo ｜ Stage 8 ｜ 关联需求：F-015（P0）｜ 粗估：2.5 人日

## 目标与价值

把事件总线（S3 已就绪的 10 类事件）聚合成进程内可直查的运营指标：问答成功率、
时延分位、阶段耗时、缓存命中、降级路由——本机零依赖排障与仪表盘数据源。

## 验收（Epic DoD）

- StatsCollector 订阅事件总线即工作，宿主一行接入（Option 或 Dependencies）。
- 构造时间序列 → 各窗口成功率/分位/计数精确可复算（单测证明公式）。
- 内存有界：任意请求量下统计内存恒定（无 per-request 保留）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.1.1 | 滚动窗口聚合器 | 8.1.1.1 stats-collector |
| 8.1.2 | 成功率与时延查询 | 8.1.1.2 stats-query |

## 依赖

- S3 事件总线；S6 指标语义参照（outcome/action/reason 枚举同源）。

## 关联设计

docs/design/04 §5/§6（指标语义）；ADR-0003（统计面与 Prometheus 面解耦）。
