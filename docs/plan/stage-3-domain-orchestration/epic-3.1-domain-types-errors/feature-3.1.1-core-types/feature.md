# Feature 3.1.1 — 公共类型

> 状态：todo ｜ Epic 3.1 ｜ 粗估：2 人日

## 目标

实现 docs/03 §2 的请求/响应类型族与 §6 的错误分类，作为 SDK 公共 API 面。

## 设计要点

- `Request{Question, Principal, Knowledge, Conversation, RequestID}`；`PrincipalScope{TenantKey, ScopeFingerprint, AuthorizationEpoch}`（三者均不进日志/指标）。
- `Response{Action, Answer, Citations, Confidence, TraceID, ConfigVersion, Degraded, Degradation}`；ActionType 四值枚举。
- 错误：ErrInvalidConfig/ErrInvalidRequest/ErrDeadlineExceeded/ErrEvidenceUnavailable/ErrDecisionUnavailable/ErrProviderContract/ErrClosed（docs/03 §6）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 3.1.1.1 | Request/Response 类型 | 3 |
| 3.1.1.2 | 错误分类 | 2 |

## 验收标准

- `go doc` 展示完整 API；每个类型有 godoc 引用 docs/03 章节。
- ActionType 非法值无法构造（构造器校验或解析时拒绝）。

## 需求关联

F-007（DecisionProvider 接口输入输出形态）、F-010（action 枚举）前置。
