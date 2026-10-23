# Stage 5 — 决策与回答

> 状态：todo ｜ 前置：S3；Epic 5.2~5.4 另需 S4 ｜ 里程碑：M5 ｜ 粗估：14 人日

## 目标

实现决策侧全部 P0 能力：DecisionProvider 契约与响应安全校验（F-007/F-010）、确定性规则引擎、Jev adapter 与降级链（F-008）、受约束决策缓存、回答组装与引用安全。Jev 私有协议只存在于 adapter 内。

## 范围内

CandidateGroup/DecisionInput/Decision 类型与校验；配置化规则求值（docs/02 rules 样例）；Jev HTTP 客户端（env 注入、单次尝试、版本捕获、响应校验）；降级链执行器（Jev→规则→决策缓存→abstain/clarify）与 degraded/reason 贯通；决策缓存 key/再校验；citation 校验与回答组装策略。

## 范围外

初判规则与决策规则引擎共享的细化策略（复用同一引擎，初判 hook 在 S3 已留）；LLM 回答生成（回答文本归宿主/上游，KAS 只做组装与校验）。

## 入口条件

- S3 完成（5.1 可开工）；S4 完成（5.2~5.4 开工）；O-2（Jev 协议文档）到位，否则 5.2 以 fake server 开发并标记 blocked 风险。

## 出口条件（DoD）

- [ ] F-007：编排器仅依赖 DecisionProvider 接口（无 Jev 类型泄漏到 orchestrator/decision 之外的包）。
- [ ] F-008：Jev timeout/5xx/非法响应场景按序降级且不超 root deadline；degraded=true + reason code 贯通 Response。
- [ ] F-010：非法 action/越界 TargetGroupID/NaN confidence 被拒（contract error）；citation ⊆ 最终有效 Evidence；无 Evidence 不 answer（默认配置）。
- [ ] docs/05 §2 决策相关全部矩阵行有自动化用例。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 5.1 | 决策契约与规则引擎 | 5.1.1 决策接口；5.1.2 规则引擎 |
| 5.2 | Jev Adapter | 5.2.1 Jev 客户端 |
| 5.3 | 降级链与决策缓存 | 5.3.1 降级编排；5.3.2 受约束决策缓存 |
| 5.4 | 回答组装与引用 | 5.4.1 引用安全 |

## 需求覆盖

F-007（5.1）、F-008（5.2+5.3）、F-010（5.1+5.4）。

## 风险与注意事项

- O-2 是最大不确定项：Jev 协议文档必须尽早索取；无文档期间用 docs/03 §3 契约构造 fake server，协议到位后改动限定在 adapter 单包。
- 降级链顺序是安全语义（docs/04 §3），不得因性能原因调整顺序。
