# Feature 1.2.2 — CI 与质量门禁

> 状态：done ｜ Epic 1.2 ｜ 粗估：1 人日

## 目标

GitHub Actions 流水线与 PR 门禁：build/test matrix、lint/vet、覆盖率上传与阈值、`-race` 常态化，让"CI 绿"成为全局 DoD 的机械判定。

## 设计要点

- CI 与 Makefile 命令同源（overview/03 §3）。
- 覆盖率阈值分级：整体基线随 Stage 递增；P0 安全路径 90%（D-8）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 1.2.2.1 | CI 流水线 | 3 |
| 1.2.2.2 | 测试基建 | 3 |

## 验收标准

- PR 触发完整流水线；main 保护规则要求全部 job 绿。
- 覆盖率报告可见且阈值失败会阻塞合并。

## 需求关联

工程基线（无 F-xxx）。
