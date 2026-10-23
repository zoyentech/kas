# ADR-0001 — v1.2 范围与不做清单

> 状态：accepted ｜ 日期：2026-10-23 ｜ 关联：docs/design/01-architecture-prd.md §1；F-001~F-014

## 背景

KAS v1.2 是首个开源版本，定位为**可嵌入宿主 Go 服务的知识问答运行时 SDK**。v1.1（内部 SDK）验证了 query runtime 的定位，v1.2 在其上补齐安全（有效期/缓存隔离）、可观测（Health/metrics/trace）与配置热替换（原子 Reload）。首版必须把边界一次定清，避免"顺手做了服务化/控制面"导致领域逻辑被基础设施耦合污染。

## 决策

**MVP = P0 需求（F-001~F-012）全部落地 + GoldenSet 验收门槛通过 + `v1.2.0` 开源发布。**

范围内：

- Go SDK 全部领域接口（docs/design/03）；五阶段精确预算与逐阶段 deadline
- 知识有效期过滤（expired/future/unknown fail-closed）与 Evidence 安全缓存
- DecisionProvider 隔离决策协议；Jev adapter 与降级链（Jev→规则→受约束决策缓存→abstain/clarify）
- 回答与引用校验（citation ∈ 最终有效 Evidence）
- Health / Prometheus / 带版本 trace；配置 Load/Validate/原子 Reload
- GoldenSet 框架与验收套件；参考用 in-memory EvidenceProvider 与故障注入设施

不做清单（v1.2 明确不做，`docs/design/01-architecture-prd.md` §1）：

- 独立服务进程与 HTTP/gRPC 包装（F-014 → post-MVP）
- 跨进程配置控制面、History/Rollback API、CLI、文件 watcher
- Redis/分布式共享缓存（F-013 → post-MVP）
- 知识导入/切片/索引/管理 UI；长期会话记忆
- 未通过 GoldenSet 前替换用户可见链路（宿主按 docs/design/06 切换阶段另行推进）

## 备选方案

| 备选 | 结论 | 理由 |
|---|---|---|
| 直接做服务进程（SDK 只做内部库） | 否决 | SDK 无状态先行，服务化只薄包装同一核心接口；先做服务会把领域逻辑与传输层耦合 |
| 内置配置 watcher + History/Rollback | 否决 | 宿主已有配置发布体系；SDK 只需 Load/Validate/Reload 原子语义，控制面归宿主 |
| 默认 Redis 共享缓存 | 否决 | 首版最小运行依赖仅宿主 + EvidenceProvider；共享缓存是 adapter 扩展（F-013），且安全 key 规则需先在进程内验证 |

## 后果

- 正面：边界清晰，Stage 1~7 的 64 人日估算可控；multica 可按 Epic 粒度并行派工。
- 负面/代价：多副本进程内缓存命中率分片；宿主需自建配置发布与 /metrics 端点。
- 缓解：命中率是观察值而非质量门槛（docs/design/05 §3）；共享缓存 adapter 留接口。

## 影响需求

| 需求 | 影响 |
|---|---|
| F-001~F-012（P0） | 全部在 v1.2 范围内，Stage 1~7 落地 |
| F-013/F-014（P1） | 明确出范围，进 post-MVP backlog（docs/plan/overview/05-post-mvp-backlog.md） |

## 附：开放决策（O-1~O-7，与 docs/plan/overview/01-mvp-scope.md §7 一致）

| # | 决策 | 状态（本 ADR 时点） | 本仓库默认处置 |
|---|---|---|---|
| O-1 | GitHub 组织与仓库名 | 待项目负责人确认 | 占位模块路径 `github.com/kas-runtime/kas`，确认后统一 sed 迁移 |
| O-2 | Jev 私有协议文档 | 待协议 Owner 提供 | Epic 5.2 以 docs/design/03 §3 契约 fake server 先行 |
| O-3 | GoldenSet 门槛数值冻结 | 待业务负责人签字 | 按 docs/design/05 §3 建议值实现，常量集中可调 |
| O-4 | 宿主 total 预算上限来源 | 待确认 | 可选配置字段 `host_max_total`，未配置则不校验 |
| O-5 | reserve 最小组装预算评审值 | 待评审 | 默认 200ms，配置可覆盖 |
| O-6 | GoldenSet 样本审批人 | 待 QA 负责人牵头 | 样本框架 + 合成样本先行，真实语料脱敏后替换 |
| O-7 | 权限判定模式 | 待确认 | 两者都支持：默认 PermissionTags 声明式映射，允许宿主注入判定函数 |
