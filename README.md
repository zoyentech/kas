# KAS — Knowledge Answer Service SDK

KAS 是**可嵌入宿主 Go 服务的知识问答运行时 SDK**：统一请求预算、Evidence 检索与知识有效期筛选、决策 provider、带引用回答、安全缓存与配置热替换。

KAS 是 query runtime，**不接管**知识事实源、导入、切片、索引管理与 UI。首版（v1.2）以 Go SDK 形态交付；后续服务化仅薄包装同一核心接口，不重写领域逻辑。

## 状态

**开发前（pre-alpha）。** 设计文件包与五级开发计划（7 Stage / 19 Epic / 30 Feature / 49 Story / 136 Task）已定稿，正在按 Stage 顺序实现。API 在 v1.2.0 发布前不保证兼容。

## 快速开始（占位）

~~~go
import kas "github.com/kas-runtime/kas"
~~~

SDK 公共 API 草案见 [docs/design/03-go-sdk-api.md](docs/design/03-go-sdk-api.md)；可运行的入门示例将在功能落地后补充。

## 文档

- [设计文件包](docs/design/) — 架构 PRD、示例配置、Go SDK API、超时/缓存/可观测、GoldenSet 验收、v1.1→v1.2 迁移
- [开发计划 WBS](docs/plan/README.md) — Stage/Epic/Feature/Story/Task 五级拆解与 multica 交付映射
- [决策记录 ADR](docs/adr/README.md) — 架构与工程决策（ADR-0001 范围、ADR-0002 技术选型）

## 徽章

<!-- CI 与 coverage 徽章在仓库推送到 GitHub 并接通 Actions 后启用（Story 1.2.2.1）。
[![CI](https://github.com/kas-runtime/kas/actions/workflows/ci.yml/badge.svg)](https://github.com/kas-runtime/kas/actions/workflows/ci.yml)
-->

## 许可

[Apache-2.0](LICENSE)。模块路径 `github.com/kas-runtime/kas` 为开放决策 O-1 的占位值，正式组织/仓库名确认后统一迁移（见 [ADR-0002](docs/adr/0002-technology-choices.md)）。
