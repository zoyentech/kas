# Story 8.5.2.1 — 熔断器

> 状态：done ｜ 作为值班运维，我希望 Jev 故障期间请求快速降级，以便整体时延不被超时预算拖垮。

## 前置依赖

- Epic 8.5.1（先保证 panic 不影响状态机）；S5 降级链。

## 任务

#### T8.5.2.1.1 状态机与接线
- **目标：** closed→open（连续失败 N，默认 5）→ half-open（冷却 30s）→ closed（探测成功）/open（再失败）。
- **产出：** `internal/decision/jev/breaker.go` + adapter 接线 + 配置节（enabled 默认 true、
  N、冷却期）。
- **步骤：** 1) 原子状态机（无锁或 mutex）；2) Decide 入口 open 即返回
  ErrDeadlineExceeded 语义错误（reason=jev_unavailable，链直接降级）；
  3) 成功/失败反馈点。
- **验收标准：** 三态迁移矩阵测试；open 期间 fake server 调用数为 0；恢复路径时延断言。
- **依赖：** 无。
- **预估：** 1d

#### T8.5.2.1.2 状态可见
- **目标：** 熔断态进 Health detail、StatsSnapshot、仪表盘卡片。
- **产出：** Health/事件/snapshot 扩展。
- **验收标准：** 8.3/8.4 消费展示（联测）。
- **依赖：** T8.5.2.1.1；Story 8.1.1.2。
- **预估：** 0.5d

## Story DoD

- [x] 熔断合入且默认开；配置可关；状态全链可见。

## 设计依据

stage-8 §关键口径（熔断口径）。
