# Story 5.4.1.1 — 引用校验

> 状态：todo ǀ 作为安全负责人，我希望引用只能来自本次最终有效证据集，以便答案可追溯且无越权泄漏。

## 前置依赖

- Epic 5.1（Decision/目标组）；S4（有效 Evidence 集）。

## 任务

#### T5.4.1.1.1 引用归属校验器
- **目标：** `ValidateCitations(final []Evidence, cites []Citation) ([]Citation, error)`：仅保留归属有效集的引用。
- **产出：** `internal/orchestrator/cite/validate.go`。
- **步骤：** 1) 按 EvidenceID(+segment) 匹配；2) 越权/过期/不在当前结果 → 丢弃并计数；3) 全部被丢弃且策略要求引用 → 转 abstain。
- **验收标准：** 混合引用集（有效+过期+未授权+未知 ID）仅有效保留；计数事件发出。
- **依赖：** 无。
- **预估：** 1d

#### T5.4.1.1.2 max_citations 截断与策略
- **目标：** 截断至 max_citations（保目标组顺序）；require_citation_for_factual_claims 强制。
- **产出：** 截断与策略检查。
- **步骤：** 1) 保序截断；2) 事实性回答无引用 → 按策略 abstain（默认）；3) 用例（1/恰上限/超上限）。
- **验收标准：** docs/02 answer 配置逐项语义有测试。
- **依赖：** T5.4.1.1.1。
- **预估：** 0.5d

#### T5.4.1.1.3 unknown 输出与无据禁答
- **目标：** output_on_unknown=abstain 生效；allow_answer_without_evidence=false 时无有效证据必不 answer。
- **产出：** 终态判定函数 + 用例。
- **步骤：** 1) 证据不足/有效期未知 → clarify 或 abstain（按 docs/01 §6"Evidence 不完整或过期时只能澄清/拒答"）；2) 纯对话模式预留配置位（默认关）。
- **验收标准：** 三种输入（无证据/全被滤/部分有效）终态正确。
- **依赖：** T5.4.1.1.2。
- **预估：** 0.5d

## Story DoD

- [ ] 引用安全三断言单测齐备（docs/05 P0 安全集的组装器层保障）。

## 设计依据

docs/01 §3/§6；docs/02 answer 节；docs/05 §2 引用完整性行。
