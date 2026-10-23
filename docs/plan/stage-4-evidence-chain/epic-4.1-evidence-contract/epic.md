# Epic 4.1 — EvidenceProvider 契约与参考实现

> 状态：done ｜ Stage 4 ｜ 关联需求：F-003 前置、工程可测试性 ｜ 粗估：3 人日

## 目标与价值

定义 Evidence 侧唯一边界（docs/03 §4）：EvidenceProvider 接口、Evidence 结构（含有效期/权限元数据）、EvidenceQuery/Result；提供契约测试套件（任何实现接入即测合规）与 in-memory 参考实现+故障注入设施，支撑 S4~S7 全部测试。

## 验收（Epic DoD）

- 契约套件覆盖：ctx 取消响应、超时映射、结果结构完整性、Health 报告。
- inmem 实现可配置返回任意 Evidence 集；故障包装器可注入 timeout/error/慢响应。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 4.1.1 | 接口与契约测试 | 4.1.1.1 契约类型；4.1.1.2 参考实现 |

## 依赖

- S3（错误分类、ctx 防护、testutil 骨架）。

## 关联设计

docs/03 §4（Evidence 合约全节）；docs/01 §3（EvidenceProvider/Evidence Cache 组件）。
