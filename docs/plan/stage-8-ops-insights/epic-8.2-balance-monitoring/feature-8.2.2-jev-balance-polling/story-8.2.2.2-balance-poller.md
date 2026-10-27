# Story 8.2.2.2 — 余额轮询器

> 状态：done ｜ 作为运维，我希望余额自动刷新并在越限时第一时间可见，以便余额耗尽前介入。

## 前置依赖

- Story 8.2.2.1。

## 任务

#### T8.2.2.2.1 后台轮询与状态
- **目标：** 周期轮询（默认 60s；失败指数退避至 5×）+ 当前态缓存 + 越限事件。
- **产出：** `internal/balance/poller.go`（New 启动、Close 停止）+ obs 事件 `BalanceEvent`。
- **步骤：** 1) 单 goroutine ticker；2) 状态迁移去重（同态不重复发事件）；
  3) unknown 态保留上次成功值 + 标记失联时长。
- **验收标准：** goleak 干净；状态迁移矩阵（normal→low→critical→unknown→恢复）测试；
  Health detail 带余额态。
- **依赖：** T8.2.2.1.1。
- **预估：** 1d

#### T8.2.2.2.2 gauge 导出与 Health 信息组件（路线 A 改造）
- **目标：** BalanceEvent → `kas_balance_amount{source,unit}`/`kas_balance_state{source}`
  gauge；Health 聚合报告含 balance 信息组件（不影响总态）。
- **产出：** obs.BalanceEvent + metrics collector 扩展 + client.Health 接线。
- **验收标准：** 采集自事件流设置 gauge；Grafana 看板余额面板可渲染（8.7 联测）。
- **依赖：** T8.2.2.2.1；Story 8.1.1.2。
- **预估：** 0.5d

## Story DoD

- [x] 轮询器合入；事件与快照数据面就绪；生命周期 -race 干净。

## 设计依据

stage-8 §关键口径；S6 事件总线。
