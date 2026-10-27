# KAS — Knowledge Answer Service SDK

[![CI](https://github.com/zoyentech/kas/actions/workflows/ci.yml/badge.svg)](https://github.com/zoyentech/kas/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

KAS 是**可嵌入宿主 Go 服务的知识问答运行时 SDK**：统一请求预算、Evidence 检索与知识有效期筛选、决策 provider、带引用回答、安全缓存与配置热替换。

KAS 是 query runtime，**不接管**知识事实源、导入、切片、索引管理与 UI。回答文本由上游决策 provider 生成，KAS 只做组装与引用安全校验。

## 安装

~~~bash
go get github.com/zoyentech/kas@v1.2.0
~~~

仓库：<https://github.com/zoyentech/kas>（O-1 已确认）。

## 三步嵌入

**1. Load** —— YAML 严格解析（未知字段拒绝）+ 全部启动校验（预算等式 F-001 等）：

~~~go
cfg, err := kas.LoadConfigFile("kas.yaml")
~~~

**2. New** —— 装配 Client：Evidence 缓存、降级链（Jev→规则→决策缓存→abstain）、安全护栏自动接线：

~~~go
client, err := kas.New(cfg, kas.Dependencies{
    EvidenceProvider: mySearchAdapter,  // 必需：适配宿主检索（实现 Search/Health）
    DecisionProvider: myDecider,        // 可选：作为降级链首选级
    PermissionChecker: kas.NewTagChecker(map[string][]string{
        "scope-vip": {"kb:vip"},        // O-7：声明式权限映射（或注入判定函数）
    }),
})
defer client.Close()
~~~

**3. Answer / Health / Reload**：

~~~go
resp, err := client.Answer(ctx, kas.Request{
    Question:  "服务支持哪些配置方式？",
    Principal: kas.PrincipalScope{TenantKey: "t", ScopeFingerprint: fp, AuthorizationEpoch: epoch},
    Knowledge: kas.KnowledgeScope{Namespace: "support-public", CorpusGeneration: "g1"},
})
// resp.Action / resp.Answer / resp.Citations / resp.ConfigVersion / resp.Degraded

health := client.Health(ctx)          // healthy | degraded | not_ready
ver, err := client.Reload(ctx, reader) // 原子替换；失败保持旧版本
~~~

完整可运行示例见 [examples/basic](examples/basic/main.go)（`go run ./examples/basic`）；配置样例见 [docs/design/02-example.yaml](docs/design/02-example.yaml)。

## 安全模型摘要

- **无据禁答**：answer 必须至少有一条引用，且 citation ⊆ 本次最终有效 Evidence（终检闸，零引用 answer 被拦截转 abstain）。
- **有效期 fail-closed**：过期/未生效/有效期未知（无批准策略）的 Evidence 一律不进答案与缓存（F-003）。
- **缓存隔离**：key 含规范化问题摘要、config_version、principal scope、授权 epoch、知识空间、内容代次与检索策略版本；存储仅保留不可逆摘要（F-004）。命中后仍按当前权限与时刻复核。
- **预算精确**：total = 初判+首轮+复判+补检+余量（±1ns 校验）；每阶段从 root 派生 deadline，provider 违约也不越过阶段预算（F-001/F-002）。
- **脱敏边界**：问题正文、Evidence 正文、身份明文、secret 不进入指标 label、常规日志与 trace（审计测试逐面扫描）。

## 可观测接入

~~~go
client, err := kas.New(cfg, deps,
    kas.WithPrometheusRegisterer(prometheus.DefaultRegisterer)) // F-006；不开 listener
~~~

指标族：`kas_requests_total`、`kas_request_duration_seconds`、`kas_stage_duration_seconds`、`kas_stage_deadline_exceeded_total`、`kas_deadline_exceeded_total`、`kas_evidence_cache_total`、`kas_evidence_cache_evictions_total`、`kas_evidence_filtered_total`、`kas_provider_errors_total`、`kas_decision_fallback_total`、`kas_health_status`（label 严格 allowlist）。Trace 经 `Dependencies.TraceSink` 注入（include 白名单 + 敏感键硬拒）。

## 监控（v1.3）

~~~go
client, _ := kas.New(cfg, deps, kas.WithPrometheusRegisterer(prometheus.DefaultRegisterer))
~~~

~~~bash
cd monitoring && docker compose up -d   # Grafana :3000（预置 KAS Overview 看板）+ Prometheus :9090
go run ./examples/metrics               # 宿主 /metrics 接入演示（含余额递减/流量）
~~~

内置：成功率/时延分位/阶段/缓存/降级指标（F-006）、余额监测四态 gauge（F-016）、
Jev 熔断（F-019，open 期快速降级）、provider panic 隔离（不击穿宿主）。
完整指南与 PromQL 速查见 [docs/monitoring.md](docs/monitoring.md)。

## 验收与门禁

~~~bash
make acceptance      # GoldenSet runner（门槛判定）+ 19 类验收矩阵 + 泄漏审计
make goldenset-freeze # P0 子集内容 hash 冻结
make audit-deps      # govulncheck + license 清单
~~~

首轮验收报告（102 条样本 / P0 30 条）：Recall@5 = 1.00（全集与 P0）、事实支持率 1.00、不可答准确率 1.00、P0 安全违规 0（[acceptance/report/report.md](acceptance/report/report.md)）。门槛数值按 docs/design/05 §3 建议值执行，待 O-3 业务冻结。

## 文档

- [设计文件包](docs/design/) — 架构 PRD、示例配置、Go SDK API、超时/缓存/可观测、GoldenSet 验收、v1.1→v1.2 迁移
- [开发计划 WBS](docs/plan/README.md) — 7 Stage / 19 Epic / 30 Feature / 49 Story / 136 Task
- [决策记录 ADR](docs/adr/README.md) — ADR-0001 范围 / ADR-0002 技术选型 / ADR-0003 v1.3 运营洞察边界（proposed）
- [v1.3 计划（Stage 8）](docs/plan/stage-8-ops-insights/stage.md) — 运营洞察与接入稳定化（路线 A：Prometheus+Grafana）
- [监控接入指南](docs/monitoring.md) — /metrics 暴露、一键监控栈、余额/熔断配置、PromQL 速查
- [CHANGELOG](CHANGELOG.md) ｜ [迁移指南](MIGRATION.md) ｜ [v1.2.0 发布说明](docs/release-notes/v1.2.0.md)
- [依赖与 License 基线](docs/dependency-license-baseline.md)

## 状态

v1.2.0 已发布（<https://github.com/zoyentech/kas/releases/tag/v1.2.0>）。API 在 v1.3 前不保证兼容。

## 许可

[Apache-2.0](LICENSE)
