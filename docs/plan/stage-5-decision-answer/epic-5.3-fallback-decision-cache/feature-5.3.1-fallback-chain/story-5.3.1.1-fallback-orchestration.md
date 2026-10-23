# Story 5.3.1.1 — 降级链执行器

> 状态：todo ǀ 作为值班运维，我希望 Jev 故障时链路自动安全降级且处处可见，以便无需人工介入也不超时。

## 前置依赖

- Epic 5.1/5.2 完成；S3 事件总线。

## 任务

#### T5.3.1.1.1 链执行器主体
- **目标：** 按序尝试各级 DecisionProvider，首个有效结果返回；全败 → abstain（带 reason=chain_exhausted）。
- **产出：** `internal/decision/fallback/chain.go`。
- **步骤：** 1) 顺序遍历（配置驱动）；2) 每级错误分类捕获不中断链；3) 返回 Decision 标注实际 Provider 与 Degraded。
- **验收标准：** docs/04 §3 顺序固定不可配置错序（order 缺级/乱序在配置校验拒绝或按注册集纠正并注明）；abstain 终态用例。
- **依赖：** 无。
- **预估：** 1d

#### T5.3.1.1.2 degraded 与 reason 贯通
- **目标：** 降级结果 → Response.Degraded=true + Degradation reason code 链（如 jev_timeout→rules）。
- **产出：** reason 汇聚逻辑 + Response 接线（编排器 rejudge 阶段）。
- **步骤：** 1) 每级失败记录 reason；2) 最终 Response.Degradation 输出受控枚举串；3) 上游错误文本/思维链不进入任何输出面（docs/04 §3）。
- **验收标准：** reason 仅枚举值；错误原文仅存内部 trace 安全字段。
- **依赖：** T5.3.1.1.1。
- **预估：** 0.5d

#### T5.3.1.1.3 降级事件与预算联动
- **目标：** 每级尝试发 FallbackEvent（route/reason/耗时）；剩余预算不足即终止进入安全终态。
- **产出：** 事件埋点 + 预算守卫用例。
- **步骤：** 1) 事件对齐 kas_decision_fallback_total 标签（route/reason）；2) 剩余预算 < 最小尝试成本时跳到终态；3) docs/05 §2 provider 阻塞行联测。
- **验收标准：** 三故障场景事件序列正确；全程不超 root deadline。
- **依赖：** T5.3.1.1.2。
- **预估：** 1d

## Story DoD

- [ ] F-008 验收证据齐备（与 5.2 fake server 联测）。

## 设计依据

docs/04 §3；docs/05 §2 降级行。
