# Story 3.3.1.2 — 内部事件总线

> 状态：done ｜ 作为 S6 实现者，我希望编排过程中的可观测事实以结构化事件流出，以便 metrics/trace 无侵入实现。

## 前置依赖

- T3.3.1.1.3（阶段串联）。

## 任务

#### T3.3.1.2.1 事件类型定义
- **目标：** 覆盖 docs/04 §5 指标与 §6 trace 所需的全部事件字段。
- **产出：** `internal/obs/event.go`（StageEvent/CacheEvent/FilterEvent/FallbackEvent/ProviderErrorEvent/HealthEvent）。
- **步骤：** 1) 按指标表逐行映射事件与字段（请求 outcome/action、阶段耗时与剩余 deadline、cache result、过滤 reason、fallback route/reason、provider error_class）；2) 字段含 config_version/trace_id；3) 显式标注禁止字段（question/身份/Evidence ID）不入事件。
- **验收标准：** §5 十类指标+§6 trace 字段逐一有对应事件字段（评审表核对留档）。
- **依赖：** T3.3.1.1.3。
- **预估：** 1d

#### T3.3.1.2.2 订阅与分发机制
- **目标：** 同步分发接口（SDK 内，避免额外 goroutine 泄漏面）+ 多订阅者注册。
- **产出：** `internal/obs/bus.go`。
- **步骤：** 1) 订阅者接口（S6 collector/trace sink 实现）；2) 分发异常隔离（单订阅者 panic 不影响主流程，记安全日志）；3) 单元测试：多订阅者按序收到全量事件。
- **验收标准：** 分发失败不影响请求结果；事件顺序与阶段执行顺序一致。
- **依赖：** T3.3.1.2.1。
- **预估：** 1d

## Story DoD

- [ ] 编排器在关键节点发事件；事件总线测试进入 CI。

## 设计依据

docs/04 §5（指标表）、§6（trace 字段表）。
