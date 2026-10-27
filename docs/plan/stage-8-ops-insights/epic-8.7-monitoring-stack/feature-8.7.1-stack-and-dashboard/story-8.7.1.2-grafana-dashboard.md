
# Story 8.7.1.2 — 预置 kas-overview 看板

> 状态：done ｜ 作为值班运维，我期望看板开箱覆盖全部关键面，以便不做 PromQL 也能值班。

## 前置依赖

- T8.7.1.1.1；Epic 8.2/8.5.2 gauge。

## 任务

#### T8.7.1.2.1 看板 JSON 与结构校验
- **目标：** kas-overview.json：健康/成功率/QPS/时延分位/阶段 p95/阶段超时/缓存命中率/
  降级路由/provider 错误/证据过滤/余额（值+状态）/熔断状态 面板齐备。
- **产出：** monitoring/grafana/dashboards/kas-overview.json + provider provisioning +
  结构校验测试（JSON 可解析、面板标题集齐、表达式非空、数据源 uid 一致）。
- **验收标准：** 校验测试进 CI；面板与 docs/monitoring.md PromQL 手册一致。
- **依赖：** 无。
- **预估：** 1d

## Story DoD

- [x] 看板合入并被 compose 自动加载。

## 设计依据

stage-8 §关键口径（PromQL 清单）。
