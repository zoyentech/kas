
# Story 8.6.1.2 — 监控栈验收

> 状态：done ｜ 作为接入方，我期望栈与看板交付即正确。

## 前置依赖

- Epic 8.7。

## 任务

#### T8.6.1.2.1 看板与栈校验
- **目标：** 看板 JSON 结构校验（面板/表达式/数据源）；compose/prometheus.yml 语法
  校验；examples/metrics 冒烟（/metrics 返回含 KAS 指标族）。
- **产出：** acceptance/matrix/monitoring_test.go。
- **验收标准：** 校验测试进 CI；compose up 手工走查留档。
- **依赖：** 无。
- **预估：** 0.5d

## Story DoD

- [x] 校验测试进 CI。

## 设计依据

ADR-0003；Story 8.7.1.2。
