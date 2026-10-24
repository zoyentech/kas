# GoldenSet 验收报告

- GoldenSet 版本（内容 hash）: `c8e22185131f0e91`
- 样本数: 102（P0 子集 30）
- 执行时间窗口: 2026-10-24T10:06:19+08:00 ~（anchor=2026-10-24T10:06:19+08:00）
- corpus/config/model: corpus-2026-10 / runner-v1 / goldenset-stub v1
- 失败用例: 0

## 指标（P0 与全集分开输出，docs/05 §4）

| 指标 | 全集 | P0 子集 | 门槛 |
|---|---|---|---|
| Evidence Recall@5 | 1.000 | 1.000 | 全集≥0.90 / P0≥0.95 |
| Precision@5 | 1.000 | 1.000 | — |
| MRR | 1.000 | 1.000 | — |
| nDCG | 1.000 | 1.000 | — |
| 引用事实支持率 | 1.000 | 1.000 | ≥0.95 |
| 不可答 abstain/clarify 准确率 | 1.000 | 1.000 | ≥0.95 |
| P0 安全违规（过期引用+forbidden 引用+无据回答） | **0** | — | = 0 |
| 时延 p50/p95/p99 (ms) | 0.0 / 0.1 / 0.3 | — | 观察值 |

## 场景覆盖

- common: 24 条
- config_switch: 10 条
- freshness: 32 条
- no_answer: 12 条
- paraphrase: 16 条
- permission: 28 条
- provider_fault: 10 条

> 人工盲评与缺陷项：占位（首轮自动化报告；人工裁定按 docs/05 §4 流程补充）。
