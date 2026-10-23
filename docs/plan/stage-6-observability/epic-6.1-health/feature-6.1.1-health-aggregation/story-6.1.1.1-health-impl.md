# Story 6.1.1.1 — Health 实现

> 状态：todo ǀ 作为运维，我希望 Health 快速且真实反映依赖状态，以便负载均衡与告警直接使用。

## 前置依赖

- S3（接口）；docs/02 health 配置解析（S2）。

## 任务

#### T6.1.1.1.1 HealthReport 类型与聚合规则
- **目标：** 类型 + 三态判定（required 失败=not_ready；optional 失败且 fallback 安全=degraded；全部正常=healthy）。
- **产出：** 根包 `health.go`；`internal/obs/health/aggregate.go`。
- **步骤：** 1) 组件状态枚举；2) 聚合函数纯逻辑；3) degraded_when_optional_provider_unavailable=false 时 optional 失败不影响总态（语义注明）。
- **验收标准：** 判定矩阵（required/optional × 正常/故障 × 开关）表驱动全绿。
- **依赖：** 无。
- **预估：** 0.5d

#### T6.1.1.1.2 并发探测与独立超时
- **目标：** 组件探测并发、各自 check_timeout；慢组件不拖垮整体。
- **产出：** `internal/obs/health/probe.go`。
- **步骤：** 1) errgroup + 每组件独立派生 ctx（复用 deadline 引擎）；2) 超时组件计 timeout 状态；3) 总耗时 ≤ check_timeout+ε 断言。
- **验收标准：** 阻塞 harness 注入下 Health 在 check_timeout 内返回；不触碰 Evidence cache/不发起查询（调用计数断言）。
- **依赖：** T6.1.1.1.1。
- **预估：** 1d

#### T6.1.1.1.3 health 指标与缓存语义
- **目标：** kas_health_status Gauge（component, status）数据面；探测结果不写 Evidence 缓存。
- **产出：** HealthEvent 接线 + 用例。
- **步骤：** 1) 事件总线发布各组件状态；2) collector 消费（6.2 实现，此处验证事件字段）；3) label 仅 component/status 枚举。
- **验收标准：** 事件字段对齐 docs/04 §5 Gauge 行；无敏感值。
- **依赖：** T6.1.1.1.2。
- **预估：** 0.5d

## Story DoD

- [ ] F-005 证据留档；Health 并发安全（纳入 S3 并发基线）。

## 设计依据

docs/03 §5；docs/04 §4；docs/05 §2 Health 行。
