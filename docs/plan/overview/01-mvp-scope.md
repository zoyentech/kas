# MVP 范围与补充决策

## 1. 定位

KAS（Knowledge Answer Service）v1.2 是**可嵌入宿主 Go 服务的知识问答运行时 SDK**：统一请求预算、Evidence 检索与知识有效期筛选、决策 provider、带引用回答、安全缓存与配置热替换。KAS 是 query runtime，**不接管**知识事实源、导入、切片、索引管理与 UI（见 docs/01 §1）。

MVP = **KAS v1.2 SDK 首版开源发布**：P0 需求（F-001~F-012）全部落地，并通过 GoldenSet 验收门槛（docs/05）。P1（F-013 共享缓存 adapter、F-014 服务化包装）进入 [post-MVP backlog](05-post-mvp-backlog.md)。

## 2. MVP 成功标准

1. P0 功能需求 F-001~F-012 全部实现且各有对应自动化测试。
2. GoldenSet 验收矩阵（docs/05 §2 全部 19 类场景）自动化跑通；P0 安全集：过期 Evidence 用于答案 = 0、跨 scope cache hit = 0、无权限 citation = 0。
3. Health / Prometheus / trace 能定位每一次 fallback 与 timeout；敏感数据（问题正文、Evidence 正文、身份明文、secret）不出现在指标、常规日志与 trace。
4. `v1.2.0` 在新开源仓库发布，含 README、godoc、示例、CHANGELOG、MIGRATION。

## 3. 范围内 / 范围外

**范围内：** Go SDK 全部领域接口（docs/03）；进程内 Evidence 缓存；确定性规则引擎；Jev adapter 与降级链；回答与引用校验；Health/Prometheus/trace；配置 Load/Validate/Reload；GoldenSet 框架与验收套件；参考用 in-memory EvidenceProvider 与故障注入设施。

**范围外（v1.2 明确不做，docs/01 §1）：** 独立服务进程与 HTTP/gRPC 包装（F-014）；跨进程配置控制面、History/Rollback API、CLI、文件 watcher；Redis/分布式缓存（F-013）；知识导入/切片/索引/管理 UI；长期会话记忆；替换用户可见链路（由宿主按 docs/06 切换阶段另行推进）。

## 4. 使用者（摘自 docs/01 §2）

| 使用者 | MVP 中的诉求 | 满足方式 |
|---|---|---|
| Go 服务开发者 | handler 调用 `Client.Answer` 得到 answer/citations/trace 版本与稳定错误 | Stage 3/5 公共 API |
| QA/产品 | 冻结 GoldenSet 对比旧链与 KAS | Stage 7 |
| 运维 | Health、超时、缓存、fallback 指标 | Stage 6 |
| 配置维护者 | Load/Validate/Reload 发布新快照 | Stage 2 |

## 5. 技术栈与工程补充决策（对设计文件的完善）

设计文件未锁定的工程决策，按以下默认值执行；与开放决策冲突时以确认结果为准：

| # | 决策点 | 默认方案 | 理由 |
|---|---|---|---|
| D-1 | 语言/版本 | Go ≥ 1.23，CI 矩阵跑最近两个大版本 | `context.WithDeadlineCause` 等需要 |
| D-2 | YAML 解析 | `gopkg.in/yaml.v3` + `KnownFields` 严格模式 | 未知字段拒绝（docs/04 §1 启动校验） |
| D-3 | 指标 | `prometheus/client_golang`，注册进宿主 Registerer | docs/04 §5 |
| D-4 | 测试断言 | 标准库 `testing` + `gotest.tools/v3`（assert 最小集） | 开源依赖最小化 |
| D-5 | License | Apache-2.0 | 专利授权，Go 生态惯例 |
| D-6 | 模块路径 | `github.com/<org>/kas`（org 见 O-1） | — |
| D-7 | 包布局 | 顶层包仅公共 API；实现全部在 `internal/`（config/cache/evidence/decision/orchestrator/obs） | 强制封装边界 |
| D-8 | 覆盖率门禁 | 核心包（internal/*）语句覆盖率 ≥ 80%，P0 安全路径（有效期过滤、引用校验、cache key）≥ 90% | 安全优先 |
| D-9 | 竞态 | 全部 CI 测试带 `-race`；并发语义测试用 `goleak` 防 context 泄漏 | F-011/F-012 |
| D-10 | 版本发布 | SemVer，`v1.2.0` tag + GitHub Release + CHANGELOG | 开源惯例 |

## 6. 需求追溯矩阵（F-xxx → Epic）

| 需求 | 优先级 | 摘要 | 落点 Epic |
|---|---|---|---|
| F-001 | P0 | 五阶段预算 total 精确相等 | 2.1 |
| F-002 | P0 | 阶段 deadline 传播不超 root/caller | 3.2 |
| F-003 | P0 | 知识有效期过滤（expired/future/unknown） | 4.2 |
| F-004 | P0 | Evidence 安全缓存 key | 4.3 |
| F-005 | P0 | Health（not_ready/degraded） | 6.1 |
| F-006 | P0 | Prometheus 指标 | 6.2 |
| F-007 | P0 | DecisionProvider 隔离决策协议 | 5.1 |
| F-008 | P0 | Jev 降级链 | 5.2 + 5.3 |
| F-009 | P0 | 版本 trace | 6.3（字段生成分散于 2.2/5.2） |
| F-010 | P0 | 安全 action 与引用 | 5.1 + 5.4 |
| F-011 | P0 | 原子 Reload | 2.2 |
| F-012 | P0 | 无状态 SDK 并发安全 | 3.3 |
| F-013 | P1 | 共享 cache adapter | post-MVP |
| F-014 | P1 | 服务化包装 | post-MVP |

首发门禁（docs/01 §7）由 Stage 7 承接：GoldenSet 门槛、P0 安全计数为零、reload 并发不串线、可定位性验证、敏感数据审计。

## 7. 开放决策（排期前需项目负责人/业务确认）

| # | 决策 | 影响 | 建议默认 |
|---|---|---|---|
| O-1 | GitHub 组织与仓库名 | Stage 1 全部任务、模块路径 | **已确认（2026-10-24）：zoyentech/kas** |
| O-2 | Jev 协议规范文档（私有协议的请求/响应 schema、错误码） | Epic 5.2 无法开工直到拿到协议文档；阻塞期可先做 5.1/5.3 | 向协议 Owner 索取；无文档则先以 docs/03 §3 契约做 fake server 开发 |
| O-3 | GoldenSet 通过门槛数值（docs/05 §3 全部为建议值） | Stage 7 验收判定 | 按建议值冻结，业务负责人签字 |
| O-4 | 宿主 total 预算上限（docs/04 §1 "total 超宿主上限"校验的数值来源） | Story 2.1.2.1 | 增加可选配置字段 `host_max_total`，未配置则不校验 |
| O-5 | reserve 最小组装预算的评审值 | Story 2.1.2.1 | 默认 200ms，配置可覆盖并记录 ADR |
| O-6 | GoldenSet 首批 100+ 样本的语料来源与去标识化审批人 | Story 7.1.1.1 | QA 负责人牵头，脱敏后冻结 |
| O-7 | 权限判定模式（宿主注入函数 vs PermissionTags 声明式映射） | Story 4.2.2.1 | 两者都支持：默认声明式映射，允许宿主注入判定函数 |

## 8. 主要风险

| 风险 | 影响 | 缓解 |
|---|---|---|
| Jev 协议文档拿不到（O-2） | Epic 5.2 延期 | 契约 fake 先行；协议到后只改 adapter 层 |
| GoldenSet 门槛未冻结（O-3） | Stage 7 无法收口 | Stage 1 期间并行推动确认，Stage 6 结束前必须冻结 |
| 有效期策略复杂度被低估（多来源 max-age、缺省策略组合） | Stage 4 延期 | 先实现 docs/02 中两种来源样例，策略表可扩展 |
| 开源合规审查发现依赖 license 冲突 | 发布延期 | D-5 Apache-2.0 + Stage 7.2.1 依赖审计提前到 Stage 1 做基线扫描 |
