# Story 1.2.1.1 — module 骨架

> 状态：todo ｜ 作为 SDK 开发者，我希望 module 与目录骨架先行确定，以便各 Stage 并行开发不撞结构。

## 前置依赖

- Epic 1.1 完成（仓库可用）；O-1（模块路径）。

## 任务

#### T1.2.1.1.1 go mod init 与目录分层
- **目标：** 初始化 module 并创建 internal 分层目录。
- **产出：** `go.mod`（模块路径 `github.com/<org>/kas`，Go ≥1.23）；`internal/{config,orchestrator,evidence,decision,cache,obs,contract}` 各含 `doc.go`（包职责一句话）。
- **步骤：** 1) `go mod init`；2) 建目录与 doc.go；3) 在 CONTRIBUTING 记录分层职责表。
- **验收标准：** `go build ./...` 通过；每个包 doc.go 描述与 overview/03 §6 测试分层一致。
- **依赖：** O-1。
- **预估：** 0.5d

#### T1.2.1.1.2 公共 API 占位
- **目标：** 顶层包放置 docs/03 公共 API 的最小占位（类型空壳、函数返回 ErrClosed 或未实现哨兵），保证编译。
- **产出：** 根包 `kas.go`（占位）等；后续 Stage 逐个替换为实现。
- **步骤：** 1) 按 docs/03 §1~§6 列 API 清单；2) 仅建空类型/签名；3) 标注 `// TODO(S2)…` 归属 Stage。
- **验收标准：** 编译通过；每个占位带 Stage 归属注释；`go doc` 可浏览。
- **依赖：** T1.2.1.1.1。
- **预估：** 1d

#### T1.2.1.1.3 Makefile 常用目标
- **目标：** 统一本地命令入口。
- **产出：** `Makefile`：`build / test / test-race / lint / cover / fmt / tidy`。
- **步骤：** 1) 目标与 CI 命令一一对应；2) `make help` 列说明。
- **验收标准：** 本地 `make` 全目标可执行，命令与 CI job 同源。
- **依赖：** T1.2.1.1.1。
- **预估：** 0.5d

## Story DoD

- [ ] module/目录/占位 API/Makefile 就位，`go build ./...` 绿。

## 设计依据

docs/03（API 雏形）；overview/01 §5 D-1/D-6/D-7。
