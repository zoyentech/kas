# Story 8.3.1.2 — JSON 端点契约

> 状态：done ｜ 作为排障工程师，我希望一次请求拿到全部关键面，以便 curl 一行看全貌。

## 前置依赖

- Story 8.3.1.1；8.1/8.2 数据面。

## 任务

#### T8.3.1.2.1 端点实现
- **目标：** `GET /api/overview`（一次拉全）、`/api/health`、`/api/stats?window=1m|5m|15m|1h`、
  `/api/balance`、`/api/config`（脱敏摘要：config_version+关键策略，不含 env 名值与 secret）。
- **产出：** `internal/diag/handlers.go`。
- **步骤：** 1) handler 复用 Client 公共方法（Stats/Health/Balance/ConfigVersion）；
  2) Content-Type/缓存头；3) 未知路径 404 JSON。
- **验收标准：** httptest 全端点契约（状态码/schema/默认窗口）；schema 与 StatsSnapshot 同源。
- **依赖：** 无。
- **预估：** 1d

## Story DoD

- [x] 端点契约测试进入 CI；schema 冻结。

## 设计依据

stage-8 §范围内；ADR-0003。
