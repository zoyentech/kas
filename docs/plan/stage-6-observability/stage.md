# Stage 6 — 可观测性

> 状态：done ｜ 前置：S3（事件总线）；可与 S4/S5 并行 ｜ 里程碑：M4（与 S4 共同达成）｜ 粗估：7 人日

## 目标

实现 Health（F-005）、Prometheus 全套指标（F-006）、trace 与版本诊断（F-009）：全部消费 S3 事件总线，无侵入；指标 label 严格限 allowlist，杜绝高基数与敏感值。

## 范围内

HealthReport 聚合（not_ready/degraded/healthy、短独立 timeout）；docs/04 §5 十类指标 collector 与宿主 Registerer 注入；TraceSink 接口、结构化 sink、include 白名单、脱敏规则、TraceID 生成与透传。

## 范围外

独立 /metrics listener（宿主提供）；链路追踪系统对接（OTel 等，post-MVP 可加 sink 适配）；模型版本进指标 label（明确禁止）。

## 入口条件

- S3 事件总线可用；组件 Health 接口（EvidenceProvider/DecisionProvider）已定义。

## 出口条件（DoD）

- [ ] F-005：required 故障 → not_ready；optional Jev 故障 + 安全 fallback → degraded；Health 不执行查询、短独立超时。
- [ ] F-006：docs/04 §5 指标表逐行实现，label 断言测试（无 question/身份/Evidence ID/trace ID/自由字符串）。
- [ ] F-009：trace 含 config_version/provider/model/model_version/fallback route 等；敏感字段零捕获（扫描测试）。
- [ ] 指标增量断言（成功/超时/cache/fallback 场景）进入 CI。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 6.1 | Health | 6.1.1 组件健康与聚合 |
| 6.2 | Prometheus 指标 | 6.2.1 collector 与宿主接入 |
| 6.3 | Trace | 6.3.1 TraceSink 与字段 |

## 需求覆盖

F-005（6.1）、F-006（6.2）、F-009（6.3）。

## 风险与注意事项

- 高基数是指标面最大风险：label allowlist 在 collector 层机械强制，禁止新增 label 绕过评审。
- Health 探测频率归宿主（docs/04 §4），SDK 侧只保证单次调用便宜且带独立超时。
