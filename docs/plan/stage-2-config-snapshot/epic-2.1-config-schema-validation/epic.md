# Epic 2.1 — 配置 schema 与校验

> 状态：done ｜ Stage 2 ｜ 关联需求：F-001（P0）｜ 粗估：4 人日

## 目标与价值

把 docs/02 YAML 与 docs/04 §1 启动校验清单实现为强类型 Config + Load/Validate API：所有非法配置在启动/Reload 即刻失败并给出字段路径，绝不带病运行。

## 验收（Epic DoD）

- `LoadConfigFile("docs/02 示例")` 成功；docs/04 §1"拒绝"清单每项都有拒绝用例。
- 错误信息格式统一：`<字段路径>: <问题>`，无 secret/全量配置回显。
- F-001 验收："启动和 Reload 拒绝不等式"由表驱动测试证明。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 2.1.1 | 配置模型与 YAML 加载 | 2.1.1.1 配置类型；2.1.1.2 Load/Validate API |
| 2.1.2 | 预算等式与参数校验 | 2.1.2.1 预算等式校验；2.1.2.2 provider/策略参数校验 |

## 依赖

- S1（module/测试基建）。

## 关联设计

docs/02（schema 全貌）；docs/03 §1（API 签名与 New 校验职责）；docs/04 §1（启动校验清单）。
