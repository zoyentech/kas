# Story 1.2.2.1 — CI 流水线

> 状态：todo ｜ 作为维护者，我希望每个 PR 自动跑完整质量检查，以便合入即可信。

## 前置依赖

- Epic 1.2 Feature 1.2.1（module/lint 就位）。

## 任务

#### T1.2.2.1.1 build+test matrix workflow
- **目标：** `.github/workflows/ci.yml` 在 Go 最近两个大版本 × linux/darwin 上 build+test。
- **产出：** `ci.yml` build/test job。
- **步骤：** 1) matrix（Go 版本、OS）；2) `make build test test-race`；3) 缓存 Go module。
- **验收标准：** 任意 PR 触发并全绿；matrix 覆盖 D-1。
- **依赖：** T1.2.1.1.3。
- **预估：** 0.5d

#### T1.2.2.1.2 lint/vet 门禁 job
- **目标：** CI 独立 job 跑 golangci-lint（固定版本）与 fmt-check。
- **产出：** `ci.yml` lint job + golangci-lint action 版本锁定。
- **步骤：** 1) 使用官方 action 固定小版本；2) 与本地配置同源。
- **验收标准：** 人为引入 lint 错误的分支被 CI 拒绝（验证一次后还原）。
- **依赖：** T1.2.1.2.1。
- **预估：** 0.5d

#### T1.2.2.1.3 覆盖率上传与阈值
- **目标：** 覆盖率报告上传（Codecov 或 artifacts）+ 阈值失败阻塞。
- **产出：** coverage job；README 徽章接通。
- **步骤：** 1) `make cover` 生成合并报告；2) 上传；3) 初始阈值 60%，写明随 Stage 递增至 80%/90% 的计划。
- **验收标准：** PR 可见覆盖率差异；低于阈值 CI 失败。
- **依赖：** T1.2.2.1.1。
- **预估：** 1d

## Story DoD

- [ ] 三类 job 全绿；main 分支保护把全部 job 设为必需。
- [ ] README 徽章显示 CI/coverage 状态。

## 设计依据

overview/01 §5 D-8/D-9；overview/03 §3。
