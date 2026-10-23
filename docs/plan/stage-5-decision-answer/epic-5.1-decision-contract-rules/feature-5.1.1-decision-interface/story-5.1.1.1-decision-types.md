# Story 5.1.1.1 — 决策类型与校验器

> 状态：done ǀ 作为架构评审者，我希望决策协议被限制在有限安全形态，以防任何上游输出直接驱动执行。

## 前置依赖

- S3 Epic 3.1（ActionType 等）。

## 任务

#### T5.1.1.1.1 决策类型族定义
- **目标：** CandidateGroup/DecisionInput/Decision/DecisionProvider 接口（docs/03 §3 逐字段）。
- **产出：** 根包 `decision.go`。
- **步骤：** 1) 类型与接口；2) Attributes 白名单过滤函数（按配置 allowlist）；3) Health(ctx) 纳入接口。
- **验收标准：** 字段映射表核对 docs/03 §3；白名单过滤用例（多余字段被剔除）。
- **依赖：** 无。
- **预估：** 1d

#### T5.1.1.1.2 统一响应校验器
- **目标：** `ValidateDecision(in DecisionInput, d Decision) error`：枚举/归属/数值三查。
- **产出：** `internal/decision/validate.go`。
- **步骤：** 1) action 枚举检查；2) TargetGroupIDs 与输入集合比对；3) Confidence NaN/Inf/越界 [0,1] 检查；4) 失败分类 ErrProviderContract。
- **验收标准：** 每类违规独立错误信息（不含上游原文）；表驱动含 docs/05 §2 非法响应行。
- **依赖：** T5.1.1.1.1。
- **预估：** 1d

#### T5.1.1.1.3 契约测试套件（决策侧）
- **目标：** 任意 DecisionProvider 实现的合规套件（校验器强制经过）。
- **产出：** `internal/contract/decision_test.go`。
- **步骤：** 1) 用例：合法决策通过；非法决策被拒且事件发出；Health 结构；ctx 取消；2) inmem 决策 provider（测试用）随套件交付。
- **验收标准：** 故意违规实现被套件抓住；Jev/规则/缓存三个实现后续均接入。
- **依赖：** T5.1.1.1.2。
- **预估：** 1d

## Story DoD

- [ ] 契约冻结；三实现（5.1.2/5.2/5.3.2）后续接入即测。

## 设计依据

docs/03 §3；docs/05 §2 决策相关行。
