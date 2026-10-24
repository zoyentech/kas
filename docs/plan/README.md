# KAS v1.2 MVP 开发计划（WBS）

**状态：** 已拆解，待评审排期。**依据：** [../design](../design) 设计文件包（01 架构 PRD / 02 示例配置 / 03 Go SDK API / 04 超时缓存可观测 / 05 GoldenSet 验收 / 06 迁移说明）。
**交付对象：** multica 团队研发。**代码仓库：** GitHub 新建开源仓库（仓库名/组织见开放决策 O-1）。

## 阅读顺序

1. [overview/01-mvp-scope.md](overview/01-mvp-scope.md) — MVP 范围、补充决策、需求追溯矩阵、开放决策
2. [overview/02-stage-plan.md](overview/02-stage-plan.md) — 7 个 Stage 划分、依赖图、里程碑与工期
3. [overview/03-wbs-conventions.md](overview/03-wbs-conventions.md) — 拆解约定、编号规则、DoD、分支/提交规范
4. [overview/04-multica-handoff.md](overview/04-multica-handoff.md) — 导入 multica 的映射与协作流程
5. 各 Stage 目录：`stage.md` → `epic-N.M-*/epic.md` → `feature-N.M.K-*/feature.md` → `story-N.M.K.S-*.md`（含 Task 卡片）

## 目录树

```text
plan/
├── README.md                       # 本文件：导航
├── overview/                       # 总览层
│   ├── 01-mvp-scope.md
│   ├── 02-stage-plan.md
│   ├── 03-wbs-conventions.md
│   ├── 04-multica-handoff.md
│   └── 05-post-mvp-backlog.md
├── stage-1-repo-foundation/        # 仓库与工程基建
│   ├── stage.md
│   ├── epic-1.1-oss-repo-collab/
│   │   ├── epic.md
│   │   ├── feature-1.1.1-repo-init/
│   │   │   ├── feature.md
│   │   │   ├── story-1.1.1.1-repo-license-baseline.md
│   │   │   └── story-1.1.1.2-collab-governance.md
│   │   └── feature-1.1.2-docs-adr/
│   │       ├── feature.md
│   │       └── story-1.1.2.1-docs-tree-adr.md
│   └── epic-1.2-go-skeleton-ci/
│       ├── epic.md
│       ├── feature-1.2.1-go-module-layout/
│       │   ├── feature.md
│       │   ├── story-1.2.1.1-module-skeleton.md
│       │   └── story-1.2.1.2-lint-format.md
│       └── feature-1.2.2-ci-quality-gates/
│           ├── feature.md
│           ├── story-1.2.2.1-ci-pipeline.md
│           └── story-1.2.2.2-test-infra.md
├── stage-2-config-snapshot/        # 配置与运行时快照
│   ├── stage.md
│   ├── epic-2.1-config-schema-validation/
│   │   ├── epic.md
│   │   ├── feature-2.1.1-config-model-yaml/
│   │   │   ├── feature.md
│   │   │   ├── story-2.1.1.1-config-types.md
│   │   │   └── story-2.1.1.2-load-validate-api.md
│   │   └── feature-2.1.2-budget-validation/
│   │       ├── feature.md
│   │       ├── story-2.1.2.1-timeout-equation.md
│   │       └── story-2.1.2.2-provider-policy-validation.md
│   └── epic-2.2-config-version-snapshot/
│       ├── epic.md
│       ├── feature-2.2.1-config-version/
│       │   ├── feature.md
│       │   └── story-2.2.1.1-canonical-hash.md
│       └── feature-2.2.2-atomic-reload/
│           ├── feature.md
│           ├── story-2.2.2.1-runtime-snapshot.md
│           └── story-2.2.2.2-reload-semantics.md
├── stage-3-domain-orchestration/   # 领域契约与编排骨架
│   ├── stage.md
│   ├── epic-3.1-domain-types-errors/
│   │   ├── epic.md
│   │   └── feature-3.1.1-core-types/
│   │       ├── feature.md
│   │       ├── story-3.1.1.1-request-response.md
│   │       └── story-3.1.1.2-error-taxonomy.md
│   ├── epic-3.2-deadline-engine/
│   │   ├── epic.md
│   │   └── feature-3.2.1-context-deadlines/
│   │       ├── feature.md
│   │       ├── story-3.2.1.1-deadline-calc.md
│   │       └── story-3.2.1.2-provider-ctx-guard.md
│   └── epic-3.3-orchestrator-skeleton/
│       ├── epic.md
│       ├── feature-3.3.1-pipeline/
│       │   ├── feature.md
│       │   ├── story-3.3.1.1-answer-flow.md
│       │   └── story-3.3.1.2-event-bus.md
│       └── feature-3.3.2-concurrency-lifecycle/
│           ├── feature.md
│           └── story-3.3.2.1-client-lifecycle.md
├── stage-4-evidence-chain/         # Evidence 链路
│   ├── stage.md
│   ├── epic-4.1-evidence-contract/
│   │   ├── epic.md
│   │   └── feature-4.1.1-evidence-interface/
│   │       ├── feature.md
│   │       ├── story-4.1.1.1-contract-types.md
│   │       └── story-4.1.1.2-reference-impl.md
│   ├── epic-4.2-validity-permission-filter/
│   │   ├── epic.md
│   │   ├── feature-4.2.1-validity-filter/
│   │   │   ├── feature.md
│   │   │   ├── story-4.2.1.1-validity-rules.md
│   │   │   └── story-4.2.1.2-filter-observability.md
│   │   └── feature-4.2.2-permission-check/
│   │       ├── feature.md
│   │       └── story-4.2.2.1-scope-permission.md
│   └── epic-4.3-evidence-cache/
│       ├── epic.md
│       ├── feature-4.3.1-normalize-cache-key/
│       │   ├── feature.md
│       │   ├── story-4.3.1.1-question-normalization.md
│       │   └── story-4.3.1.2-cache-key.md
│       └── feature-4.3.2-cache-policy/
│           ├── feature.md
│           ├── story-4.3.2.1-memory-cache.md
│           └── story-4.3.2.2-read-write-path.md
├── stage-5-decision-answer/        # 决策与回答
│   ├── stage.md
│   ├── epic-5.1-decision-contract-rules/
│   │   ├── epic.md
│   │   ├── feature-5.1.1-decision-interface/
│   │   │   ├── feature.md
│   │   │   └── story-5.1.1.1-decision-types.md
│   │   └── feature-5.1.2-rules-engine/
│   │       ├── feature.md
│   │       └── story-5.1.2.1-rule-evaluation.md
│   ├── epic-5.2-jev-adapter/
│   │   ├── epic.md
│   │   └── feature-5.2.1-jev-client/
│   │       ├── feature.md
│   │       ├── story-5.2.1.1-jev-protocol.md
│   │       └── story-5.2.1.2-jev-validation.md
│   ├── epic-5.3-fallback-decision-cache/
│   │   ├── epic.md
│   │   ├── feature-5.3.1-fallback-chain/
│   │   │   ├── feature.md
│   │   │   └── story-5.3.1.1-fallback-orchestration.md
│   │   └── feature-5.3.2-constrained-decision-cache/
│   │       ├── feature.md
│   │       └── story-5.3.2.1-decision-cache.md
│   └── epic-5.4-answer-assembly/
│       ├── epic.md
│       └── feature-5.4.1-citation-safety/
│           ├── feature.md
│           ├── story-5.4.1.1-citation-validation.md
│           └── story-5.4.1.2-answer-builder.md
├── stage-6-observability/          # 可观测性
│   ├── stage.md
│   ├── epic-6.1-health/
│   │   ├── epic.md
│   │   └── feature-6.1.1-health-aggregation/
│   │       ├── feature.md
│   │       └── story-6.1.1.1-health-impl.md
│   ├── epic-6.2-prometheus-metrics/
│   │   ├── epic.md
│   │   └── feature-6.2.1-collector/
│   │       ├── feature.md
│   │       ├── story-6.2.1.1-metric-families.md
│   │       └── story-6.2.1.2-host-integration.md
│   └── epic-6.3-tracing/
│       ├── epic.md
│       └── feature-6.3.1-trace-sink/
│           ├── feature.md
│           └── story-6.3.1.1-trace-impl.md
└── stage-7-acceptance-release/     # GoldenSet 验收与发布
    ├── stage.md
    ├── epic-7.1-goldenset/
    │   ├── epic.md
    │   ├── feature-7.1.1-cases-runner/
    │   │   ├── feature.md
    │   │   ├── story-7.1.1.1-case-schema-samples.md
    │   │   └── story-7.1.1.2-runner-scoring.md
    │   └── feature-7.1.2-acceptance-matrix/
    │       ├── feature.md
    │       ├── story-7.1.2.1-safety-scenarios.md
    │       ├── story-7.1.2.2-degradation-scenarios.md
    │       └── story-7.1.2.3-concurrency-scenarios.md
    └── epic-7.2-release/
        ├── epic.md
        ├── feature-7.2.1-security-review/
        │   ├── feature.md
        │   └── story-7.2.1.1-release-audit.md
        └── feature-7.2.2-v1.2.0-release/
            ├── feature.md
            └── story-7.2.2.1-docs-release.md
```

## 规模统计

| 层级 | 数量 | 说明 |
|---|---:|---|
| Stage | 7 | MVP 阶段，串/并行关系见 overview/02 |
| Epic | 19 | 对应 multica Issue 粒度 |
| Feature | 30 | 对应 multica 子 Issue / 标签粒度 |
| Story | 49 | 对应 multica 带 stage 的任务 |
| Task | 135 | 最小执行单元，每张任务卡 0.5~2 人日 |

## 使用约定（摘要，详见 overview/03）

- 所有任务卡包含：目标 / 产出 / 步骤 / 验收标准 / 依赖 / 预估。
- 全局 DoD：单测通过、`-race` 通过、CI 绿、无 PII/secret 泄漏、关联 F-xxx 需求可追溯。
- 本阶段只输出方案，不含任何实现代码；代码在新开源仓库中按本计划实施。
