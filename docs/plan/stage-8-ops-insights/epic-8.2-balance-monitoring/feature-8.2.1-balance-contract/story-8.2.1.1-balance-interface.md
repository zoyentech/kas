# Story 8.2.1.1 — BalanceSource 接口与四态

> 状态：done ｜ 作为宿主开发者，我希望余额判定独立可注入，以便非 Jev 计费源也能接入同一监测面。

## 前置依赖

- 无。

## 任务

#### T8.2.1.1.1 接口与状态机
- **目标：** `BalanceSource`（`Balance(ctx) (BalanceInfo, error)`）+ 四态判定纯函数 + 配置节。
- **产出：** `internal/balance/source.go`、`config.BalanceConfig`（low/critical 阈值、interval、enabled）。
- **步骤：** 1) 类型（金额、币种单位、刷新时刻、来源名）；2) `Classify(amount, cfg)` 四态；
  3) 配置校验（0 ≤ critical < low；interval > 0）。
- **验收标准：** 阈值边界表驱动（==low 归 low 等边界明确）；非法配置拒绝。
- **依赖：** 无。
- **预估：** 0.5d

## Story DoD

- [x] 契约冻结；判定矩阵单测全绿。

## 设计依据

stage-8 §关键口径（余额口径）。
