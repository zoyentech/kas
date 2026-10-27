# Story 8.2.2.1 — Jev 余额 adapter（fake 先行）

> 状态：todo ｜ 作为 SDK，我希望余额协议封装在一个 adapter 内，以便 O-2 文档到位后只改一处。

## 前置依赖

- Story 8.2.1.1；O-2（未到位按最小契约 + fake server）。

## 任务

#### T8.2.2.1.1 余额端点适配
- **目标：** Jev 余额 HTTP 查询（端点 env 注入、超时、脱敏）+ fake server 扩展余额路由。
- **产出：** `internal/balance/jev.go`；`testutil.JevFakeServer` 增 `GET /balance`。
- **步骤：** 1) `balance_endpoint_env` 配置；2) 复用 jev 包 http client 口径（单次、env 注入、
  错误不含值）；3) fake 可脚本化余额与错误（200/5xx/慢）。
- **验收标准：** env 缺失构造期处理与决策 adapter 一致；错误串零余额值零 key。
- **依赖：** 无。
- **预估：** 1d

## Story DoD

- [ ] adapter 契约测试（fake 驱动正常/5xx/超时/畸形体）全绿。

## 设计依据

Story 5.2.1.1（env/脱敏同口径）；ADR-0003。
