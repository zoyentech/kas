# Feature 5.1.2 — 确定性规则引擎

> 状态：todo ｜ Epic 5.1 ｜ 粗估：1.5 人日

## 目标

配置驱动的确定性规则求值：条件完整可审计、输出有限 action、优先级稳定（docs/04 §3），同时作为初判 hook 的实现载体与降级链第二级。

## 设计要点

- 内置条件（docs/02 样例）：`valid_evidence_count_equals_zero`、`decision_confidence_below_threshold`；条件注册表可扩展。
- 命中输出：action + reason（规则 name）；多命中按声明序优先级稳定。
- 纯函数：同输入同输出，无副作用，便于 trace 解释。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 5.1.2.1 | 规则求值 | 3 |

## 验收标准

- docs/05 §2"多条规则冲突 → 优先级稳定、trace 可解释"自动化。
- 规则引擎同时可挂初判 hook（S3 接口）演示。

## 需求关联

F-008（降级链第二级）、F-007（实现 DecisionProvider）。
