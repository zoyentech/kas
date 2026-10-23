# Epic 7.1 — GoldenSet 框架与验收

> 状态：done ｜ Stage 7 ｜ 关联需求：docs/01 §7 首发门禁（全部 P0 的验收面）｜ 粗估：7 人日

## 目标与价值

建立可重复、可冻结、可对比的验收资产：GoldenSet 用例与 runner、评分与报告、19 类矩阵场景自动化。不以单次模型 judge 作为验收（docs/05 §1）。

## 验收（Epic DoD）

- `acceptance/` 目录可独立运行（make acceptance）输出报告产物。
- P0 安全集单独统计且全零断言；报告含版本快照（GoldenSet 版本/样本数/corpus/config/model version/时间窗口/指标/缺陷项）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 7.1.1 | 用例与运行器 | 7.1.1.1 用例 schema 与样本；7.1.1.2 runner 与评分 |
| 7.1.2 | 验收矩阵自动化 | 7.1.2.1 安全场景；7.1.2.2 降级场景；7.1.2.3 并发场景 |

## 依赖

- S1~S6 全部；O-3/O-6 关闭。

## 关联设计

docs/05 全文（用例结构/矩阵/门槛/产物）。
