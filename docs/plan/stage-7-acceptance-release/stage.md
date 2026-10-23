# Stage 7 — GoldenSet 验收与发布

> 状态：todo ｜ 前置：S1~S6 ｜ 里程碑：M6 ｜ 粗估：10 人日

## 目标

把 docs/05 验收设计变成可重复执行的自动化资产：用例 schema 与 ≥100 条去标识化样本、runner 与评分器、19 类验收矩阵场景套件、P0 安全断言；完成发布前安全/依赖审计与 v1.2.0 开源发布。

## 范围内

GoldenSet JSON schema 与样本集构建（含 P0 子集冻结流程）；runner（固定 corpus generation/授权上下文/valid_at）；评分器（Recall@k/Precision@k/MRR/nDCG、回答事实、引用质量、时延分位）；验收矩阵场景套件（安全/降级/并发三组）；报告产物；敏感数据与依赖审计；README/godoc/CHANGELOG/MIGRATION 定稿；v1.2.0 tag + GitHub Release。

## 范围外

与宿主旧链的线上对比（shadow run 属宿主接入，post-MVP）；门槛数值的重新商定（O-3 冻结后只执行）。

## 入口条件

- S1~S6 出口条件全部满足；O-3（门槛冻结）、O-6（样本审批）关闭。

## 出口条件（DoD）

- [ ] docs/05 §2 验收矩阵 19 类场景全部自动化且在 CI 可重复执行。
- [ ] P0 安全集断言全零（过期引用/跨 scope hit/未授权 citation/无据回答）。
- [ ] 建议门槛（docs/05 §3）逐项出具报告：Recall@5 全集 ≥0.90、P0 ≥0.95、引用事实支持率 ≥0.95、不可答 abstain/clarify ≥0.95 等（以 O-3 冻结值为准）。
- [ ] 发布审计通过：敏感数据零泄漏扫描、依赖漏洞与 license 检查。
- [ ] v1.2.0 发布：tag、Release Notes、CHANGELOG、README/godoc/MIGRATION 完整，发布产物归档 `docs/release-notes/`。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 7.1 | GoldenSet 框架与验收 | 7.1.1 用例与运行器；7.1.2 验收矩阵自动化 |
| 7.2 | 发布与文档 | 7.2.1 安全与合规审查；7.2.2 v1.2.0 发布 |

## 需求覆盖

全部 P0 需求的最终验收面（docs/01 §7 首发门禁）。

## 风险与注意事项

- 样本质量决定验收效度：去标识化审批（O-6）与覆盖度检查（docs/05 §1 场景分布）不可省。
- 门槛未达标时不得以放宽断言方式"通过"；未过门槛旧链继续对用户（docs/05 §4），KAS 仅影子比较。
