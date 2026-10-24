# Story 1.2.1.2 — 静态检查与格式化

> 状态：done ｜ 作为代码评审者，我希望机械问题交给 lint 工具，以便 review 聚焦设计正确性。

## 前置依赖

- T1.2.1.1.1（module 存在）。

## 任务

#### T1.2.1.2.1 golangci-lint 配置与基线
- **目标：** 提交 `.golangci.yml` 并零告警。
- **产出：** `.golangci.yml`。
- **步骤：** 1) 启用 linters：errcheck、govet、staticcheck、ineffassign、unused、misspell、gocritic、revive、copyloopvar；2) 关闭与本项目冲突的规则并注明理由；3) 仓库当前代码零告警。
- **验收标准：** `golangci-lint run` 退出码 0；配置文件每条豁免有注释理由。
- **依赖：** T1.2.1.1.1。
- **预估：** 0.5d

#### T1.2.1.2.2 格式化规则统一
- **目标：** gofmt/goimports 统一，CI 校验。
- **产出：** Makefile `fmt`/`fmt-check` 目标；`.editorconfig`。
- **步骤：** 1) `gofumpt` 作为更强格式化基线（若团队同意则固定版本）；2) CI 增加 fmt-check。
- **验收标准：** `make fmt` 后 `git diff` 为空；CI fmt-check 通过。
- **依赖：** T1.2.1.2.1。
- **预估：** 0.5d

## Story DoD

- [x] lint/format 全绿并纳入 CI（由 1.2.2.1 接线）。

## 设计依据

overview/03 §3 DoD（CI 绿含 vet/lint）。
