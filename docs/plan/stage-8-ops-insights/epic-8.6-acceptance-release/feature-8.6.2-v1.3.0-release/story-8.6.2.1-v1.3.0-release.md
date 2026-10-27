# Story 8.6.2.1 — v1.3.0 发布

> 状态：done ｜ 作为首个使用运营能力的用户，我希望文档带我启用全部新能力。

## 前置依赖

- Epic 8.6.1；Epic 8.1~8.5。

## 任务

#### T8.6.2.1.1 文档定稿
- **目标：** CHANGELOG v1.3.0 段（feat(stats/balance/diag/dashboard/stability)）；README
  新章节（运营能力一览 + 指南链接）；godoc 走查（新导出符号零 TODO）。
- **产出：** 三处文档更新。
- **验收标准：** 外部视角 30 分钟标准走查通过。
- **依赖：** 无。
- **预估：** 0.5d

#### T8.6.2.1.2 tag 发布
- **目标：** `git tag v1.3.0` 推送 → 发版流水线（validate/test/acceptance/release/预热）全绿。
- **产出：** v1.3.0 Release。
- **验收标准：** 流水线 5 job 全绿；go get @v1.3.0 可安装。
- **依赖：** T8.6.2.1.1。
- **预估：** 0.5d

## Story DoD

- [x] 发布在线；本计划 Stage 8 全部置 done。

## 设计依据

stage-7 发版模式（release.yml 复用）。
