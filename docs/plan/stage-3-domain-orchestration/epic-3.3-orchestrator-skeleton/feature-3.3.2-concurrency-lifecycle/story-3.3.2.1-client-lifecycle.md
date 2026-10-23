# Story 3.3.2.1 — Client 生命周期与并发基线

> 状态：todo ｜ 作为宿主，我希望 SDK 并发安全且关闭语义明确，以便随宿主副本水平扩容与优雅停机。

## 前置依赖

- Feature 3.3.1（主流程）；S2（Reload）。

## 任务

#### T3.3.2.1.1 Close 语义实现
- **目标：** `Client.Close()`：拒新请求（ErrClosed）、关自有资源、幂等。
- **产出：** `internal/orchestrator/close.go` + 根包 Client.Close。
- **步骤：** 1) closed 标志（atomic）与各入口检查；2) 自有资源清单化关闭（事件订阅、进程内 cache 停止）；3) 幂等测试；4) godoc 声明宿主资源归属（docs/03 §6 末段）。
- **验收标准：** Close 后 Answer/Reload 返回 ErrClosed；二次 Close 无 panic；在途请求自然完成或被其 ctx 取消。
- **依赖：** 无。
- **预估：** 1d

#### T3.3.2.1.2 混合并发基线测试
- **目标：** Answer/Health/Reload/Close 混合并发压力测试（`-race`）。
- **产出：** `internal/orchestrator/concurrency_test.go`。
- **步骤：** 1) N goroutine 混合调用；2) 断言无 panic、无版本串线（配合 S2 测试）；3) CI 常跑 + `-count=10` 抽测。
- **验收标准：** goleak 干净；错误仅来自预期类别（ErrClosed/Deadline）。
- **依赖：** T3.3.2.1.1。
- **预估：** 1d

## Story DoD

- [ ] F-012 证据留档；并发基线进入 CI。

## 设计依据

docs/03 §6（并发安全/Close/宿主资源）；docs/01 §6（无状态与扩容）。
