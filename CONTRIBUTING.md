# 贡献指南

感谢参与 KAS！本文件描述开发环境、分支/提交约定与质量门禁。开发计划的完整拆解约定见 [docs/plan/overview/03-wbs-conventions.md](docs/plan/overview/03-wbs-conventions.md)。

## 开发环境

- Go ≥ 1.23（CI 矩阵跑最近两个大版本；`context.WithDeadlineCause` 等特性必需）
- `make build test test-race lint cover fmt` 与 CI 命令一一对应（见 `Makefile`）
- 依赖最小化：仅允许 `gopkg.in/yaml.v3`（D-2）、`prometheus/client_golang`（D-3）、`gotest.tools/v3`（D-4，仅测试）与 `go.uber.org/goleak`（D-9，仅测试）四类运行时/测试依赖；新增依赖需先提 ADR

## 分支与提交

- 分支：`feat/S<Stage>-<slug>`（如 `feat/S2-reload-semantics`）、`fix/S…`、`chore/S…`、`docs/S…`
- 提交：Conventional Commits（`feat(config): …`、`fix(cache): …`），正文引用任务卡 ID（如 `T4.3.2.1.2`）
- 合并：PR 必须关联至少一个 Story/Task ID；squash merge 到 main；main 分支保护要求 CI 全绿 + 至少 1 人 review
- 每个 Story 一个 PR 或少量 PR；禁止跨 Epic 的大 PR

## 质量门禁（全局 DoD）

每个 Story 完成必须满足：

1. `go build ./...`、`go vet`、golangci-lint、`go test -race ./...` 全绿；覆盖率达 D-8 阈值（核心包 ≥ 80%，安全路径 ≥ 90%）。
2. 新逻辑有对应单元测试；安全路径（有效期过滤、引用校验、cache key、降级链）覆盖 ≥ 90%。
3. 错误信息含字段路径，**不含** secret、问题正文、身份明文。
4. 公共 API 变更有 godoc 注释；行为变更同步更新计划文档状态。
5. 关联需求 ID（F-xxx）在 Story 文档回填。

## 包布局与分层职责

公共 API 只在顶层包 `kas`；实现全部在 `internal/`（D-7，强制封装边界）：

| 包 | 职责 |
|---|---|
| `internal/config` | 配置类型、YAML 严格解析、启动校验、config_version、原子 Reload |
| `internal/orchestrator` | 请求编排、逐阶段 deadline、事件总线、Client 生命周期 |
| `internal/evidence` | EvidenceProvider 适配、有效期/权限过滤 |
| `internal/decision` | DecisionProvider、规则引擎、Jev adapter、降级链 |
| `internal/cache` | 问题规范化、安全 cache key、有界内存缓存 |
| `internal/obs` | Health 聚合、Prometheus collector、TraceSink |
| `internal/contract` | EvidenceProvider/DecisionProvider/Cache 接口合规契约套件（实现方一行接入自测） |
| `internal/testutil` | fake clock、临时配置构造器、goleak 挂载等测试工具（仅测试引用） |
| `acceptance/` | GoldenSet 用例 schema、样本、runner 与验收矩阵（Stage 7） |

## 测试约定

- 断言库：`gotest.tools/v3`（最小集：`assert`/`fs`/`golden`，不引入 BDD 框架）
- 并发测试：一律可被 `-race` 复现；用 `goleak` 防 goroutine/context 泄漏（各包 `TestMain` 挂 `testutil.CheckLeaks`）
- 属性测试：规范化保留否定词/数字/单位/型号符号（`internal/cache/normalize_test.go`）
- 契约测试：新实现 Evidence/Decision/Cache 时，在自身 `_test.go` 中调用 `contract.Suite…` 一行接入合规套件
- 敏感性：任何测试不得写入真实 PII/secret；样例问题文本必须去标识化

## 行为准则

参与本项目即同意遵守 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。
