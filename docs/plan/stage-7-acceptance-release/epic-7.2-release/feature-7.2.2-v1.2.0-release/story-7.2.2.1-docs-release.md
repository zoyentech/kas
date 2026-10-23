# Story 7.2.2.1 — 文档与发布

> 状态：done ǀ 作为首个外部用户，我希望文档能带我零支持跑通，以便快速评估采用。

## 前置依赖

- Epic 7.2.1 审计通过；Epic 7.1 报告归档。

## 任务

#### T7.2.2.1.1 README 完整化与快速上手
- **目标：** README 从骨架到完整：定位、安装、三步嵌入（New→Answer→Health）、示例配置、可观测接入、安全模型摘要、文档索引。
- **产出：** `README.md` + `examples/`（最小可运行示例，使用 inmem provider）。
- **步骤：** 1) 按目标结构撰写；2) 示例代码进 CI 编译验证；3) 外部视角走查（30 分钟标准）。
- **验收标准：** 示例 `go run` 可跑通；README 与 docs/plan 链接一致。
- **依赖：** 无。
- **预估：** 1d

#### T7.2.2.1.2 godoc/CHANGELOG/MIGRATION 定稿
- **目标：** 公共 API godoc 审阅修正；CHANGELOG（v1.2.0 条目按 Conventional Commits 汇总）；MIGRATION.md（docs/06 定稿版）。
- **产出：** 三份文档。
- **步骤：** 1) `go doc -all` 走查补注释；2) CHANGELOG 自动+人工整理；3) MIGRATION 迁移五节（配置/代码/热替换/切换/回退）落成。
- **验收标准：** godoc 无 TODO 残留于导出符号；MIGRATION 覆盖 docs/06 全部章节。
- **依赖：** 无。
- **预估：** 1d

#### T7.2.2.1.3 v1.2.0 tag 与 Release
- **目标：** 打 tag、发布 GitHub Release（说明+验收报告+审计报告链接）。
- **产出：** `v1.2.0` tag；Release 页面；`docs/release-notes/v1.2.0.md` 归档。
- **步骤：** 1) 门槛核对清单（M6 门禁逐项）；2) tag（签名可选）与 Release 草稿评审；3) 发布并同步 multica 收口。
- **验收标准：** Release 附件/链接完整；`go get github.com/<org>/kas@v1.2.0` 可用。
- **依赖：** T7.2.2.1.1、T7.2.2.1.2。
- **预估：** 0.5d

## Story DoD

- [ ] MVP 完成：发布在线、报告归档、multica 全部 Story 置 done。

## 设计依据

docs/01 §7；docs/06（迁移素材）；overview/01 §2（MVP 成功标准）。
