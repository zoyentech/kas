# 拆解约定与编号规则

## 1. 层级定义

| 层级 | 含义 | 粒度 | 文件形态 |
|---|---|---|---|
| Stage | MVP 阶段，有明确入口/出口条件与里程碑 | 1~3 周 | `stage-N-*/stage.md` |
| Epic | 交付一项连贯能力的大块工作 | 多 feature，1~2 周 | `epic-N.M-*/epic.md` |
| Feature | 一个可独立验收的功能单元 | 1~3 人日 | `feature-N.M.K-*/feature.md` |
| Story | 用户/开发者视角的价值切片，含验收标准 | 0.5~2 人日 | `story-N.M.K.S-*.md` |
| Task | 最小执行单元，一张卡可独立分派、独立验收 | 0.5~2 人日 | Story 文件内 `#### T…` 任务卡 |

编号：`Stage.Epic.Feature.Story.Task`，如 `4.3.2.1` 的第 3 张卡记作 `T4.3.2.1.3`。目录名带英文 slug（kebab-case），正文用中文。

## 2. 任务卡模板

```markdown
#### T<N.M.K.S.T> 标题
- **目标：** 一句话说清做什么
- **产出：** 代码位置 / 文件 / 测试 / 文档（如 internal/cache/lru.go + lru_test.go）
- **步骤：** 1) … 2) … 3) …
- **验收标准：** 可判定的检查项（每条可被测试或评审验证）
- **依赖：** T… / 无 / 外部：O-2
- **预估：** 0.5d / 1d / 2d
```

## 3. 全局 DoD（每个 Story 完成必须满足）

1. 代码合入 main 且 CI 绿（build / vet / lint / test -race / coverage 达 D-8 阈值）。
2. 新逻辑有对应单元测试；安全路径（有效期过滤、引用校验、cache key、降级链）测试覆盖 ≥ 90%。
3. 错误信息含字段路径，不含 secret、问题正文、身份明文（F-009/§6 安全约束）。
4. 公共 API 变更有 godoc 注释；行为变更同步更新本计划文档状态。
5. 关联需求 ID（F-xxx）在 Story 文档"需求关联"处回填可追溯。

## 4. 状态模型

`todo → in_progress → in_review → done`；`blocked`（注明阻塞原因与解除条件）。Story 内每张任务卡用 `[ ] / [x]` 勾选。Stage 状态在其 `stage.md` 头部维护。

## 5. 分支与提交约定

- 分支：`feat/S<E>-<slug>`（如 `feat/S2-reload-semantics`）、`fix/S…`、`chore/S…`、`docs/S…`。
- 提交：Conventional Commits（`feat(config): …`、`fix(cache): …`），正文可引用任务卡 ID（`T4.3.2.1.2`）。
- 合并：PR 必须关联至少一个 Story/Task ID；squash merge 到 main；main 分支保护要求 CI 绿 + 至少 1 人 review。
- 每个 Story 一个 PR 或少量 PR；禁止跨 Epic 的大 PR。

## 6. 测试分层

| 层 | 内容 | 位置 |
|---|---|---|
| 单元测试 | 各包函数级，含表驱动与错误路径 | 同包 `_test.go` |
| 契约测试 | EvidenceProvider/DecisionProvider/Cache 接口的可复用合规套件 | `internal/contract/` |
| 并发测试 | Reload/Answer/Health 并发、snapshot 不串线 | `internal/orchestrator/` 与 Stage 7 场景 |
| 属性测试 | 问题规范化保留否定词/数字/单位/型号符号 | `internal/cache/normalize_test.go` |
| 验收套件 | GoldenSet 19 类场景 + P0 安全断言 | `acceptance/`（Stage 7） |

## 7. 设计文件引用约定

任务卡"设计依据"指向 `docs/0x…md §n`（`docs/design/` 设计文件包）。实现与设计冲突时：以设计文件为准；设计文件需修订时先提 ADR 再改代码。

## 8. 估算标尺

`0.5d` 纯编码+单测半天；`1d` 编码+单测+文档一天；`2d` 含复杂测试（并发/属性/契约）或外部联调。评审与返工已含在 Story 预估汇总中，约上浮 20%。
