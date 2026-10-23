# ADR-0002 — 技术选型（D-1~D-10）

> 状态：accepted ｜ 日期：2026-10-23 ｜ 关联：docs/plan/overview/01-mvp-scope.md §5

## 背景

设计文件包（docs/design）锁定了领域行为，但未锁定语言版本、解析库、指标库、包布局与质量门禁等工程决策。这些决策影响每个 Stage 的代码形态，必须在 Stage 1 一次性定清。

## 决策

| # | 决策点 | 决策 | 理由 |
|---|---|---|---|
| D-1 | 语言/版本 | Go ≥ 1.23，CI 矩阵跑最近两个大版本 | `context.WithDeadlineCause` 等需要 |
| D-2 | YAML 解析 | `gopkg.in/yaml.v3` + `KnownFields` 严格模式 | 未知字段拒绝（docs/design/04 §1 启动校验） |
| D-3 | 指标 | `prometheus/client_golang`，注册进宿主 Registerer | docs/design/04 §5；KAS 不开 listener |
| D-4 | 测试断言 | 标准库 `testing` + `gotest.tools/v3` 最小集 | 开源依赖最小化 |
| D-5 | License | Apache-2.0 | 专利授权，Go 生态惯例 |
| D-6 | 模块路径 | `github.com/<org>/kas`（org 见 O-1） | 本仓库暂用占位 `github.com/kas-runtime/kas` |
| D-7 | 包布局 | 顶层包仅公共 API；实现全部在 `internal/` | 强制封装边界 |
| D-8 | 覆盖率门禁 | 核心包语句覆盖率 ≥ 80%，P0 安全路径 ≥ 90% | 安全优先 |
| D-9 | 竞态 | 全部 CI 测试带 `-race`；并发测试用 `goleak` 防 context 泄漏 | F-011/F-012 |
| D-10 | 版本发布 | SemVer，`v1.2.0` tag + GitHub Release + CHANGELOG | 开源惯例 |

并发测试另加 `go.uber.org/goleak`（仅测试依赖）。

## 备选方案

| 决策点 | 备选 | 结论 | 理由 |
|---|---|---|---|
| D-1 | Go 1.22 | 否决 | 无 `WithoutCancel`/`WithDeadlineCause`，deadline 引擎表达力不足 |
| D-2 | `sigs.k8s.io/yaml` / `goccy/go-yaml` | 否决 | 前者经 JSON 中转丢严格性；后者生态与稳定性不及 yaml.v3 |
| D-2 | 宽松解析（忽略未知字段） | 否决 | 拼写错误的配置字段会静默失效，违背 fail-fast 校验目标 |
| D-3 | 自研轻量指标 + expvar | 否决 | 宿主生态以 Prometheus 为主，自研增加接入成本 |
| D-4 | testify | 否决 | 断言面更大但带来依赖惯例分裂；gotest.tools/v3 与 go test 风格一致 |
| D-7 | 平铺包（实现也在顶层） | 否决 | 无法用编译器强制封装边界，公共面会失控 |
| D-9 | 仅 CI 带 -race | 否决 | F-011（原子 Reload）/F-012（并发安全）要求本地开发也默认竞态检测 |
| D-10 | 不做 tag/Release，仅 main | 否决 | 开源使用者需要可寻址版本；GoldenSet 验收需固定版本对照 |

## 后果

- 正面：严格模式把配置错误前移到 Load；`internal/` 布局让公共 API 面可评审；`-race`+goleak 让并发语义从第一天被测。
- 负面/代价：yaml.v3 维护节奏放缓（已知 CVE 已修复在 v3.0.1）；依赖 client_golang 引入 prometheus/common 传递依赖。
- 缓解：`go mod tidy` 锁定 + Stage 7.2.1 依赖 license/漏洞审计（基线扫描已在 Stage 1 建立，见 docs/plan/overview/01 §8）。

## 影响需求

| 需求/任务 | 影响 |
|---|---|
| F-001/F-011 | D-2 严格模式 + D-9 race 是验收前提 |
| F-005/F-006/F-009 | D-3 指标库与宿主 Registerer 注入方式 |
| 全部 Stage | D-6/D-7 决定目录与 import 路径；D-8 决定 CI 阈值曲线（60% → 80%/90%） |
