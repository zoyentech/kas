# Epic 5.2 — Jev Adapter

> 状态：done ｜ Stage 5 ｜ 关联需求：F-007/F-008/F-009（P0）｜ 粗估：3.5 人日

## 目标与价值

Jev 私有协议封装在单一 adapter 包内：env 注入凭证、请求映射、单次尝试（max_attempts=1）、响应结构校验（复用统一校验器）、model/model_version 捕获进 Decision 与 trace。协议细节不泄漏到包外（F-007）。

## 验收（Epic DoD）

- 契约套件（5.1.1.1）全绿；超时/5xx/非法响应映射到正确错误类别并触发降级（联 5.3.1）。
- endpoint/api_key 从 env 读取，任何日志/错误/trace 不含 api_key 值。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 5.2.1 | Jev 客户端 | 5.2.1.1 协议封装；5.2.1.2 响应校验与版本捕获 |

## 依赖

- Epic 5.1；O-2（Jev 协议文档；无文档时以 fake server 开发）。

## 关联设计

docs/03 §3（Jev 约束：默认一次、私有协议隔离）；docs/02 decision.jev 节；docs/04 §3（降级链第一级）。
