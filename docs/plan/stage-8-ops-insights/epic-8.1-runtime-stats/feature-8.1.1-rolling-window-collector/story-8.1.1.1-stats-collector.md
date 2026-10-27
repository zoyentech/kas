# Story 8.1.1.1 — StatsCollector 聚合器

> 状态：todo ｜ 作为值班运维，我希望进程内直接查最近窗口的成功率与时延分位，以便不依赖 Prometheus 快速排障。

## 前置依赖

- S3 事件总线（已完成）。

## 任务

#### T8.1.1.1.1 窗口数据结构
- **目标：** 环形桶（1m/5m/15m/1h，桶宽 5s）+ 每桶计数族与对数时延直方图。
- **产出：** `internal/stats/window.go`。
- **步骤：** 1) 桶结构（requests/success/degraded/failed/outcome 计数、duration 直方图引用）；
  2) 桶过期回收（惰性，读取时清扫）；3) 时延对数桶边界表（1ms~10s ~30 桶+溢出）。
- **验收标准：** 桶翻转正确（fake clock ±1 桶）；内存恒定（写入 10^6 次后桶数有界）。
- **依赖：** 无。
- **预估：** 1d

#### T8.1.1.1.2 事件→聚合映射
- **目标：** 10 类事件写入窗口（RequestEnd 主轴；StageEnd 阶段直方图；Cache/Filter/Fallback 计数）。
- **产出：** `internal/stats/collector.go`（实现 obs.Subscriber）。
- **步骤：** 1) Notify 分派；2) 成功率口径落地（stage.md 关键口径）；3) ProviderError 按
  provider/error_class 聚 top-N（固定槽位，无自由字符串增长）。
- **验收标准：** 构造事件序列 → 计数逐项相等；-race 并发写读干净。
- **依赖：** T8.1.1.1.1。
- **预估：** 1d

#### T8.1.1.1.3 接线与生命周期
- **目标：** New 自动订阅（Option `WithStats` 默认开）；Close 不需要（纯同步订阅）。
- **产出：** 根包 Option + client.go 接线；godoc。
- **验收标准：** examples/basic 打开后即可查询到统计。
- **依赖：** T8.1.1.1.2。
- **预估：** 0.5d

## Story DoD

- [ ] 聚合器合入；窗口/直方图数学有公式级单测；-race 干净。

## 设计依据

本计划 stage-8 §关键口径；docs/design/04 §5（指标语义同源）。
