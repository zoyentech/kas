# Feature 7.1.1 — 用例与运行器

> 状态：todo ｜ Epic 7.1 ｜ 粗估：3.5 人日

## 目标

GoldenSet 用例 schema、样本集构建流程与 runner/评分器/报告产物。

## 设计要点

- 用例字段（docs/05 §1）：case_id、risk、question、principal_scope、knowledge_namespace、corpus_generation、valid_at、expected_action、expected_evidence_ids、forbidden_evidence_ids、expected_answer_facts、must_abstain、tags。
- runner 固定：同一 corpus generation、授权上下文与 valid_at（docs/05 §4）。
- 评分：Evidence Recall@k/Precision@k/MRR/nDCG（有排名标注时）、回答正确/漏答/无据陈述、引用覆盖/准确/过期率、p50/p95/p99、stage timeout、cache hit、fallback 比例。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 7.1.1.1 | 用例 schema 与样本 | 3 |
| 7.1.1.2 | runner 与评分 | 3 |

## 验收标准

- 样本 ≥100 条、覆盖 docs/05 §1 场景分布；P0 子集冻结留档。

## 需求关联

首发门禁。
