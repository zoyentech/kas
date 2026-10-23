# Feature 4.1.1 — Evidence 接口与契约测试

> 状态：todo ｜ Epic 4.1 ｜ 粗估：3 人日

## 目标

EvidenceProvider 契约定型 + 可复用合规测试套件 + 参考实现，使宿主接入与 SDK 测试共用同一套期望。

## 设计要点

- `Evidence{ID, SourceID, Title, Excerpt, ValidFrom, ValidUntil, ObservedAt, ContentVersion, PermissionTags, Score}`（docs/03 §4）。
- `Search(ctx, EvidenceQuery) (EvidenceResult, error)` + `Health(ctx)`。
- 契约套件模式沿用 T1.2.2.2.3。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 4.1.1.1 | 契约类型与套件 | 2 |
| 4.1.1.2 | 参考实现与故障注入 | 2 |

## 验收标准

- 契约套件本身有自测（用 inmem 实现跑通全套）。
- Evidence 元数据字段缺省语义（ValidFrom/ValidUntil 为 nil = 未知）有明确定义并测试。

## 需求关联

F-003（元数据供过滤）、F-004（元数据供 TTL）。
