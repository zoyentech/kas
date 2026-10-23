# Feature 5.2.1 — Jev 客户端

> 状态：todo ｜ Epic 5.2 ｜ 粗估：3.5 人日

## 目标

Jev HTTP 客户端 adapter：协议请求/响应映射、单次尝试、传输超时接线（≤ 阶段预算）、响应校验与版本捕获。

## 设计要点

- 配置：endpoint_env/api_key_env/model/model_version/max_attempts（docs/02）。
- fake Jev server（testutil）先行：协议未定稿时可开发/测试全部行为。
- transport timeout = min(阶段剩余, 配置)，绝不自建更宽超时（docs/04 §1）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 5.2.1.1 | 协议封装 | 3 |
| 5.2.1.2 | 响应校验与版本捕获 | 2 |

## 验收标准

- fake server 全场景（合法/timeout/5xx/非法结构）驱动测试通过。
- adapter 包对外仅暴露 DecisionProvider 实现。

## 需求关联

F-007/F-008/F-009。
