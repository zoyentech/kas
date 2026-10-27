
# Story 8.6.1.1 — 稳定性与余额验收

> 状态：done ｜ 作为验收负责人，我期望 panic/熔断/余额行为被场景级固化。

## 前置依赖

- Epic 8.5、8.2。

## 任务

#### T8.6.1.1.1 panic 注入矩阵
- **目标：** evidence/decision/preliminary/assembler 四注入点全走查：请求降级、
  error_class=panic 事件、进程存活、-race。
- **产出：** acceptance/matrix/stability_test.go。
- **验收标准：** 四场景断言通过。
- **依赖：** 无。
- **预估：** 0.5d

#### T8.6.1.1.2 熔断与余额场景
- **目标：** 熔断 open 期零 Jev 调用+快速降级+半开恢复；余额四态迁移+unknown 保留+
  gauge/事件对账。
- **产出：** 同套件扩展。
- **验收标准：** 场景断言通过；-race。
- **依赖：** 无。
- **预估：** 0.5d

## Story DoD

- [x] 套件进 CI；GoldenSet 不回归。

## 设计依据

stage-8 §出口条件。
