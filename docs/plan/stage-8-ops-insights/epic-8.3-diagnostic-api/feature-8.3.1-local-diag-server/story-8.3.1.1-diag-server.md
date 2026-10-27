# Story 8.3.1.1 — 诊断服务器与生命周期

> 状态：done ｜ 作为宿主开发者，我希望一行启用/关闭诊断面，以便默认零侵入、排障时秒开。

## 前置依赖

- 无（端点数据依赖 8.1/8.2，可先骨架后填充）。

## 任务

#### T8.3.1.1.1 server 骨架与 Option
- **目标：** `WithDiagnosticServer(addr string)`；空 addr 默认 `127.0.0.1:0`（随机端口），
  `Client.DiagAddr() string` 返回实际监听地址。
- **产出：** `internal/diag/server.go` + 根包 Option。
- **步骤：** 1) mux 构建；2) 启动/Close（http.Server Shutdown，超时 2s）；3) 不启用时零开销。
- **验收标准：** 启用→监听→Close→拒绝连接；goleak 干净；-race。
- **依赖：** 无。
- **预估：** 0.5d

#### T8.3.1.1.2 静态资源挂载
- **目标：** `/` 服务内嵌 SPA（8.4 产出，先占位页）。
- **产出：** `internal/diag/static.go`（embed 接线）。
- **验收标准：** 浏览器打开 `/` 得 HTML。
- **依赖：** T8.3.1.1.1。
- **预估：** 0.5d

## Story DoD

- [x] 生命周期干净；默认关闭零侵入有测试证明（无 goroutine 无端口）。

## 设计依据

ADR-0003。
