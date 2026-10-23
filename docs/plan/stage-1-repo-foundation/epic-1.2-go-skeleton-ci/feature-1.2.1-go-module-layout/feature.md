# Feature 1.2.1 — module 与包结构

> 状态：done ｜ Epic 1.2 ｜ 粗估：1 人日

## 目标

初始化 Go module 与分层包结构，形成 Stage 2~7 的落码位置约定。

## 设计要点

- 目录布局（D-7）：顶层包 `kas/`（或模块根）只放公共 API；`internal/config`、`internal/orchestrator`、`internal/evidence`、`internal/decision`、`internal/cache`、`internal/obs`、`internal/contract`。
- 顶层公共 API 以 docs/03 草案为唯一雏形（命名实现期可调，见 docs/03 开头说明）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 1.2.1.1 | module 骨架 | 3 |
| 1.2.1.2 | 静态检查与格式化 | 2 |

## 验收标准

- 空实现编译通过；`internal/` 分层就位并写入 CONTRIBUTING。
- lint 配置提交且零告警基线。

## 需求关联

工程基线（无 F-xxx）。
