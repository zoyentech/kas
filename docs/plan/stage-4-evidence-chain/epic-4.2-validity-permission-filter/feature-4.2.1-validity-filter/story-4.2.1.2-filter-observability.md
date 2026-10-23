# Story 4.2.1.2 — 过滤可观测

> 状态：todo ｜ 作为运维，我希望每次过滤都有 reason 计数，以便发现知识库新鲜度问题。

## 前置依赖

- Story 4.2.1.1；T3.3.1.2.2（事件总线）。

## 任务

#### T4.2.1.2.1 过滤事件与 reason 归因
- **目标：** 过滤发生即发 FilterEvent（reason ∈ {expired, future, unauthorized, unknown_validity}，数量）。
- **产出：** filter 层事件埋点 + 单测。
- **步骤：** 1) 聚合单请求各 reason 计数；2) 事件只含计数与 reason，绝不含 Evidence 内容/ID（docs/04 §5 label 约束）；3) S6 collector 消费预接线验证。
- **验收标准：** 构造混合过滤场景，事件计数准确；事件 payload 中无 Evidence 标识。
- **依赖：** T4.2.1.1.1。
- **预估：** 1d

## Story DoD

- [ ] kas_evidence_filtered_total 四类 reason 数据面就绪（S6 只做消费）。

## 设计依据

docs/04 §5（kas_evidence_filtered_total）；docs/04 §6（过滤数量入 trace）。
