# Story 5.2.1.2 — 响应校验与版本捕获

> 状态：done ǀ 作为安全评审者，我希望 Jev 输出经过与任何 provider 相同的安全校验，以便信任边界统一。

## 前置依赖

- Story 5.2.1.1；T5.1.1.1.2（统一校验器）。

## 任务

#### T5.2.1.2.1 响应结构校验与 contract error 分类
- **目标：** Jev 响应 → Decision 的严格解析 + 统一校验器 + 错误分类。
- **产出：** `internal/decision/jev/parse.go`。
- **步骤：** 1) 严格 JSON 解析（未知字段拒绝或忽略策略注明）；2) 复用 ValidateDecision；3) 违规 → ErrProviderContract + provider_errors 事件（error_class=contract）。
- **验收标准：** docs/05 §2 非法 action/越界 ID/NaN 三用例在 Jev 路径通过（fake server 注入）。
- **依赖：** T5.2.1.1.2。
- **预估：** 1d

#### T5.2.1.2.2 model/model_version 捕获
- **目标：** 响应中的模型标识捕获进 Decision（F-009 trace 数据源）。
- **产出：** 解析字段 + 透传。
- **步骤：** 1) 捕获 model/model_version（响应优先，回退配置声明）；2) 值仅进 Decision/trace，不进 metric label（docs/04 §5 高基数禁令）；3) 用例。
- **验收标准：** Decision.Model/ModelVersion 正确；指标面无版本 label（S6 联测）。
- **依赖：** T5.2.1.2.1。
- **预估：** 0.5d

## Story DoD

- [x] Jev 实现接入决策契约套件全绿。

## 设计依据

docs/03 §3（Decision 字段）；docs/04 §5（版本不进 label）、§6（版本进 trace）。
