# Epic 6.2 — Prometheus 指标

> 状态：done ｜ Stage 6 ｜ 关联需求：F-006（P0）｜ 粗估：3.5 人日

## 目标与价值

实现 docs/04 §5 十类指标 collector，注册进宿主提供的 prometheus.Registerer（SDK 不开 listener）；事件总线 → collector 接线；label allowlist 机械强制。

## 验收（Epic DoD）

- 指标表逐行实现并有增量断言测试（成功/超时/cache/fallback/过滤场景）。
- 高基数防护：任何含自由字符串的 label 注入在测试层被拒（allowlist 校验）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 6.2.1 | collector 与宿主接入 | 6.2.1.1 指标族；6.2.1.2 宿主接入与增量断言 |

## 依赖

- S3（事件总线）；D-3（client_golang）。

## 关联设计

docs/04 §5（指标表与禁令）；docs/01 §3（Health/metrics/trace 组件）；docs/02 metrics 节。
