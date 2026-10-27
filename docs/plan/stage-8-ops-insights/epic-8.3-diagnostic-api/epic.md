# Epic 8.3 — 诊断查询 API

> 状态：todo ｜ Stage 8 ｜ 关联需求：F-017（P0）｜ 粗估：1.5 人日

## 目标与价值

可选本机 HTTP JSON API：一次性拉全或分面查询（overview/health/stats/balance/config）。
默认**不启用**（Option 开启），默认绑定 127.0.0.1，无鉴权——本机单用户诊断用途，
边界决策见 ADR-0003。

## 验收（Epic DoD）

- `WithDiagnosticServer` 启用后端点可用；默认关闭零侵入（不启 server 无任何 goroutine/端口）。
- JSON schema 与 StatsSnapshot 同源稳定；httptest 契约测试。
- Close 后连接拒绝。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.3.1 | 本机诊断服务 | 8.3.1.1 diag-server；8.3.1.2 diag-endpoints |

## 依赖

- Epic 8.1/8.2 产物（stats/balance 数据）；ADR-0003。

## 关联设计

ADR-0003（可选工具组件边界）；docs/design/01 §1（不做对外服务——本组件不改变该边界）。
