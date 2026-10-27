# Story 8.5.1.1 — deadline.Call panic 守卫

> 状态：todo ｜ 作为宿主服务负责人，我希望三方 provider 的 bug 不至于拖垮我的进程，以便接入无底线担忧。

## 前置依赖

- 无。

## 任务

#### T8.5.1.1.1 recover 与错误映射
- **目标：** provider goroutine panic → 捕获 → `ErrProviderContract`（含阶段名，不含 panic 值原文——
  panic 值可能含敏感数据，仅计数进事件 error_class=panic）。
- **产出：** `internal/deadline/guard.go` 修改 + 单测。
- **步骤：** 1) goroutine 内 defer recover；2) 错误语义（与超时/取消区分）；
  3) 编排器 emitProviderError 增 panic 分类。
- **验收标准：** 注入 panic 的 evidence/decision provider：请求降级返回、进程存活、
  -race 干净、事件计数正确。
- **依赖：** 无。
- **预估：** 1d

#### T8.5.1.1.2 契约套件扩展
- **目标：** EvidenceProviderSuite/DecisionProviderSuite 增 panic 用例（合规实现不得击穿调用方）。
- **产出：** internal/contract 扩展。
- **验收标准：** 违约 panic 实现被套件判失败而非崩溃测试进程。
- **依赖：** T8.5.1.1.1。
- **预估：** 0.5d

## Story DoD

- [ ] panic 注入测试进 CI；godoc 声明 panic 隔离语义。

## 设计依据

v1.2.0 审核差距 #4；docs/design/04 §1。
