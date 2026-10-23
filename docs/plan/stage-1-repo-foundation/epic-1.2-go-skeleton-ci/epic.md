# Epic 1.2 — Go 工程骨架与 CI

> 状态：done ｜ Stage 1 ｜ 关联需求：工程基线（无 F-xxx）｜ 粗估：2 人日

## 目标与价值

建立 Go module 骨架、静态检查与 CI 质量门禁，让后续每个 Stage 的代码从第一天起就在统一的构建/测试/覆盖率约束下开发，PR 门禁自动生效。

## 验收（Epic DoD）

- `go build ./...`、`go vet`、golangci-lint、`go test -race ./...` 在 CI 全绿。
- 覆盖率上传并达基线（初始 ≥ 60%，随 Stage 递增至 D-8 目标）。
- 包布局符合 D-7（公共 API 在顶层包，实现在 internal/）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 1.2.1 | module 与包结构 | 1.2.1.1 module 骨架；1.2.1.2 静态检查与格式化 |
| 1.2.2 | CI 与质量门禁 | 1.2.2.1 CI 流水线；1.2.2.2 测试基建 |

## 依赖

- Epic 1.1（仓库存在）。

## 关联设计

docs/03（包结构面向公共 API 的雏形）；overview/01 §5 D-1/D-4/D-7/D-8/D-9。
