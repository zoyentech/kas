# Stage 3 — 领域契约与编排骨架

> 状态：done ｜ 前置：S2 ｜ 里程碑：M3 ｜ 粗估：9 人日

## 目标

落地 docs/03 全部公共领域类型与哨兵错误；五阶段 deadline 引擎（root 派生、阶段独立派生、全路径 cancel、provider ctx 合规）；编排器骨架与内部事件总线（S6 消费）；Client 并发安全与 Close 语义。骨架注入 fake provider 可走通"初判→首轮→复判→补检→余量"全流程。

## 范围内

Request/Response/ActionType/Scope/Citation 类型；ErrXxx 错误分类；deadline 计算与派生规则（docs/04 §1 context 规则）；编排主流程骨架（各阶段挂载点）；StageEvent 事件总线；并发/Close 测试。

## 范围外

Evidence 真实检索与过滤（S4）；决策/降级/回答组装（S5）；Health/metrics/trace 实现（S6，仅事件埋点）。

## 入口条件

- S2 出口条件满足（snapshot/版本可用）。

## 出口条件（DoD）

- [ ] F-002：所有阶段 deadline 不晚于 root/caller deadline；阻塞 provider 被 ctx 取消（误差 ≤50ms）用测试证明。
- [ ] F-012：Answer/Health/Reload 并发 `-race` 通过；Close 后调用返回 ErrClosed。
- [ ] fake provider 端到端骨架测试：五阶段按序执行、事件按序发出、每请求 config_version 恒定。
- [ ] 阶段间禁止派生（从上一阶段 context 派生下一阶段）有防护与测试。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 3.1 | 领域类型与错误 | 3.1.1 公共类型 |
| 3.2 | 五阶段 deadline 引擎 | 3.2.1 context deadline 规则 |
| 3.3 | 编排器骨架 | 3.3.1 阶段流水线；3.3.2 并发与生命周期 |

## 需求覆盖

F-002（3.2）、F-012（3.3）、F-007 前置（DecisionProvider 接口形态，3.1 类型）。

## 风险与注意事项

- deadline 语义是全链安全基础（docs/04 §1 专门强调"不从上一阶段派生"），必须防护+测试双保险，不能只靠约定。
- 事件总线是 S6 的唯一数据面，字段设计要覆盖 docs/04 §5/§6 全部指标与 trace 需求，避免 S6 返工。
