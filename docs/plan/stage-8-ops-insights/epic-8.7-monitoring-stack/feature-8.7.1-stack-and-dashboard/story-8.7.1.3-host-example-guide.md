
# Story 8.7.1.3 — 宿主示例与监控指南

> 状态：done ｜ 作为宿主开发者，我期望有可运行样例与指南，以便 30 分钟接完监控。

## 前置依赖

- T8.7.1.1.1、T8.7.1.2.1。

## 任务

#### T8.7.1.3.1 examples/metrics 可运行演示
- **目标：** 本地起 fake Jev（decide+balance）+ inmem 证据 + /metrics:8080 + 持续流量，
  看板即时出数（含余额递减演示状态迁移、可注入 Jev 故障演示熔断）。
- **产出：** examples/metrics/main.go（CI 编译验证）。
- **验收标准：** go run 后栈中看板出数。
- **依赖：** 无。
- **预估：** 1d

#### T8.7.1.3.2 docs/monitoring.md 指南
- **目标：** 宿主暴露 /metrics（WithPrometheusRegisterer+promhttp 代码）、起栈、看板导览、
  PromQL 手册（成功率/分位/降级/余额/熔断）、保留期与告警归宿主说明。
- **产出：** docs/monitoring.md + README 链接。
- **验收标准：** 外部视角 30 分钟标准走查。
- **依赖：** 无。
- **预估：** 0.5d

## Story DoD

- [x] 示例与指南合入；README 监控章节就位。

## 设计依据

F-015/F-017/F-018。
