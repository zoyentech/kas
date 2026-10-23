# Epic 5.4 — 回答组装与引用

> 状态：todo ｜ Stage 5 ｜ 关联需求：F-010（P0）｜ 粗估：3 人日

## 目标与价值

回答安全组装：引用只能指向最终有效 Evidence（拒绝越权/过期/不在当前结果中的引用）；max_citations 截断；无有效引用不得给出事实性回答（allow_answer_without_evidence=false 默认）；有效期不足/未知 → clarify/abstain（docs/01 §3 回答组装器）。

## 验收（Epic DoD）

- docs/05 §2"引用完整性：所有 citation 属于最终有效 Evidence"P0 断言自动化。
- output_on_unknown=abstain、require_citation_for_factual_claims 默认策略生效。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 5.4.1 | 引用安全 | 5.4.1.1 引用校验；5.4.1.2 回答组装器 |

## 依赖

- Epic 5.1（Decision）；S4（最终有效 Evidence 集来源）。

## 关联设计

docs/01 §3（回答组装器）；docs/02 answer 节；docs/03 §2（引用约束）；docs/05 §2 引用完整性行。
