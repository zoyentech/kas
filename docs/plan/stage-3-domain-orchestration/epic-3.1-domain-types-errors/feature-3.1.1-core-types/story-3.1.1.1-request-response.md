# Story 3.1.1.1 — Request/Response 类型

> 状态：done ｜ 作为宿主开发者，我希望请求/响应结构与 docs/03 一致且语义自解释，以便 handler 集成无歧义。

## 前置依赖

- S1 占位 API（T1.2.1.1.2）替换。

## 任务

#### T3.1.1.1.1 Request 与 Scope 类型
- **目标：** Request/PrincipalScope/KnowledgeScope/ConversationContext 落地。
- **产出：** 根包 `types_request.go`。
- **步骤：** 1) 按 docs/03 §2 定义；2) godoc 标注 TenantKey/ScopeFingerprint 的隐私约束（不进日志/指标）；3) Request 校验函数（Question 非空、RequestID 格式）。
- **验收标准：** 空问题 Request 被校验拒绝（ErrInvalidRequest）；字段与 docs/03 一致。
- **依赖：** 无。
- **预估：** 0.5d

#### T3.1.1.1.2 Response 与 ActionType
- **目标：** Response/ActionType/Degradation 落地；Degradation 为受控 reason code（非自由文本）。
- **产出：** 根包 `types_response.go`；reason code 常量表。
- **步骤：** 1) ActionType 四枚举；2) Degradation 取值定义为常量（如 `jev_unavailable`、`decision_cache_hit`）；3) Confidence 为 *float64（可空）。
- **验收标准：** 非法 action 字符串解析拒绝；reason code 集中一处定义并注释对应 docs/04 §3。
- **依赖：** T3.1.1.1.1。
- **预估：** 0.5d

#### T3.1.1.1.3 Citation/EvidenceRef 类型
- **目标：** 引用结构（指向 Evidence 的稳定引用，含 EvidenceID/SourceID/锚点）。
- **产出：** 根包 `types_citation.go`。
- **步骤：** 1) 定义 Citation（EvidenceID、SourceID、可选 segment）；2) 预留 S5 引用校验所需字段；3) godoc 说明"citation 必须属于最终有效 Evidence"（docs/03 §2）。
- **验收标准：** 结构可序列化（JSON round-trip 测试），为 S5 校验留足字段。
- **依赖：** T3.1.1.1.2。
- **预估：** 0.5d

## Story DoD

- [ ] 类型族落地，`go vet`/godoc 检查通过；S4/S5 可开始面向接口开发。

## 设计依据

docs/03 §2 全节。
