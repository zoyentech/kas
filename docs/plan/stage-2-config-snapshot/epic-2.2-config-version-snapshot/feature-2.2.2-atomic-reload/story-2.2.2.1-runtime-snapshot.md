# Story 2.2.2.1 — Runtime Snapshot 构建

> 状态：todo ｜ 作为 SDK，我希望一次请求的全部行为参数来自同一不可变快照，以便版本一致性与可测试性。

## 前置依赖

- Epic 2.1；Feature 2.2.1（版本生成）。

## 任务

#### T2.2.2.1.1 snapshot 结构与构建器
- **目标：** 定义不可变 snapshot（Config、config_version、派生预算表、provider 引用）与构建函数。
- **产出：** `internal/runtime/snapshot.go`。
- **步骤：** 1) 字段全部不可变（私有+构造器）；2) 构建时做 New 级同步校验（docs/03 §1 清单）；3) 暴露只读访问器。
- **验收标准：** 构建后无 setter；校验失败返回 ErrInvalidConfig 聚合错误。
- **依赖：** T2.2.1.1.3。
- **预估：** 1d

#### T2.2.2.1.2 New() 集成与 Dependencies 校验
- **目标：** `New(cfg, deps, opts...)` 组装 Client：必要依赖校验（EvidenceProvider 必需、Jev 可选）、opts 注入（registry、TraceSink 等 S6 用占位接口）。
- **产出：** 根包 `client.go` 的 New 骨架；`internal/runtime/build.go`。
- **步骤：** 1) deps 完整性校验（nil/ErrInvalidConfig）；2) opts 机制（functional options，S6 实现具体项）；3) 构建 snapshot 并加载为 current。
- **验收标准：** 缺 EvidenceProvider 时 New 报错；docs/03 §1 校验清单逐项有用例。
- **依赖：** T2.2.2.1.1。
- **预估：** 1d

## Story DoD

- [ ] snapshot/New 可用；"准备 provider/runtime"步骤有独立函数供 Reload 复用。

## 设计依据

docs/01 §3（Runtime Manager）、§6（发布顺序）；docs/03 §1（New 同步校验）。
