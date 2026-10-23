# Epic 6.3 — Trace

> 状态：done ｜ Stage 6 ｜ 关联需求：F-009（P0）｜ 粗估：2 人日

## 目标与价值

TraceSink 接口与默认结构化 sink：记录 docs/04 §6 全部安全字段（trace_id、config_version、provider/model/model_version、cache 结果、fallback route、阶段耗时与剩余 deadline、候选/过滤计数、action、reason code），严格脱敏（无密钥/问题正文/prompt/Evidence 正文/身份明文）。

## 验收（Epic DoD）

- include 白名单（docs/02 tracing 节）生效：capture_question_text=false、capture_evidence_text=false 默认且默认配置下零敏感字段。
- TraceID 生成、Response 透传、事件可关联（同一 trace_id 贯穿请求）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 6.3.1 | TraceSink 与字段 | 6.3.1.1 trace 实现 |

## 依赖

- S3（事件总线、TraceSink 占位接口）。

## 关联设计

docs/03 §6（TraceSink 接口）；docs/04 §6（字段与脱敏）；docs/02 tracing 节。
