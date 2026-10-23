# Stage 1 — 仓库与工程基建

> 状态：todo ｜ 前置：无 ｜ 里程碑：M1 ｜ 粗估：4 人日

## 目标

GitHub 开源仓库从零到 CI 绿：仓库与许可、协作治理文件、设计文档迁移与 ADR 机制、Go module 骨架、lint/CI/覆盖率门禁、测试基建。后续所有 Stage 在此仓库上开发。

## 范围内

开源仓库初始化（含分支保护）、治理文件（CONTRIBUTING/SECURITY/CODEOWNERS/模板）、docs 目录与 ADR 流程、Go module 与 `internal/` 分层骨架、golangci-lint、GitHub Actions（build/test/lint/coverage/-race）。

## 范围外

任何业务代码（配置/编排等自 S2 起）；README 完整化（S7 发布时定稿）；release 自动化（S7）。

## 入口条件

- O-1（组织/仓库名）与 D-5（License）确认。

## 出口条件（DoD）

- [ ] 仓库公开可见，main 分支保护（CI 绿 + ≥1 review）生效。
- [ ] LICENSE、CONTRIBUTING、CODE_OF_CONDUCT、SECURITY.md、CODEOWNERS、PR/Issue 模板齐备。
- [ ] 设计文件包迁移至 `docs/design/`，ADR-0001（v1.2 范围）与 ADR-0002（技术选型）归档。
- [ ] `go build ./... && go test -race ./...` 在 CI 全绿；覆盖率基线上传。
- [ ] 空 API 占位编译通过，包布局符合 D-7。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 1.1 | 开源仓库与协作基建 | 1.1.1 仓库初始化与许可；1.1.2 文档结构与决策记录 |
| 1.2 | Go 骨架与 CI | 1.2.1 module 与包结构；1.2.2 CI 与质量门禁 |

## 需求覆盖

无直接 F-xxx；为全部 P0 需求提供工程与合规基线。

## 风险与注意事项

- O-1 未决会阻塞 1.1.1 全部任务，需最先关闭。
- 依赖 license 基线扫描提前做（overview/01 §8 风险缓解），避免 S7 发布时返工。
