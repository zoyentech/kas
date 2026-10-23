# multica 团队交付对接说明

## 1. WBS → multica 映射

| 本计划层级 | multica 载体 | 命名规范 |
|---|---|---|
| Stage | 里程碑 / 阶段标签 `S1`~`S7` | 按 overview/02 依赖关系设置前置 |
| Epic | Issue（父） | `[S4][E4.3] Evidence 安全缓存` |
| Feature | 子 Issue 或 Issue 标签 `E4.3/F4.3.2` | `[S4][E4.3][F4.3.2] 缓存策略实现` |
| Story | 带分阶段（stage）的任务 | 标题前缀 Story ID，如 `[4.3.2.1] 有界内存缓存` |
| Task | 任务卡逐条导入为 task item | 保留 T 编号，验收标准原样粘贴 |

导入顺序：先建 S1~S7 里程碑 → 按 Epic 建 Issue（正文粘 `epic.md`）→ 建 Feature 子项（粘 `feature.md`）→ Story 任务（粘 story 正文与任务卡）。任务卡"验收标准"直接作为 task 的完成判据，不得弱化。

## 2. 排期建议

- 串行主线 S1→S2→S3；S3 后并行两条线：线 A = S4→S5，线 B = S6（汇合于 S7）。
- Epic 5.1 可提前到 S4 期间做（只依赖 S3）。Epic 7.1.1 样本构建与代码解耦，可从 S4 期间启动。
- 每条线一个 owner；跨线共享类型变更（Stage 3 产出）需在线 A/B 开工前冻结。

## 3. 工作流约定

1. 领取 Story → 状态 `in_progress` → 按 Task 卡实施 → 全部卡勾选且 DoD 满足 → `in_review` 提 PR。
2. PR 标题引用 Story ID；描述附测试证据（覆盖数、关键断言输出）。
3. Review 通过、CI 绿、squash 合入后 Story 置 `done`；Epic 下全部 Feature 完成后 Epic 收口。
4. 阻塞（如 O-2 Jev 协议文档未到位）立即置 `blocked` 并注明依赖 ID，不静默绕过；禁止通过关闭有效期校验或放大 deadline 让测试变绿（docs/06 §6.B）。
5. 每完成一个 Stage，负责人核对 `stage.md` 出口条件并留档（PR 列表 + 验证输出）。

## 4. 验收与发布门禁

- M1~M5 里程碑验收 = 对应 stage.md 出口条件逐项核对。
- M6（v1.2.0 发布）门禁 = docs/01 §7：GoldenSet 达 docs/05 冻结门槛；过期 Evidence 使用为 0；跨 scope cache hit 为 0；reload 并发版本不串线；Health/Prometheus/trace 可定位 fallback 与 timeout；敏感数据不出现在指标与常规日志。
- 未过门禁不 tag 发布；发布产物（报告、版本快照）归档到仓库 `docs/release-notes/`。

## 5. 沟通约定

- 所有讨论引用稳定 ID（F-xxx / T…），不使用口头指代。
- 设计变更走 ADR（repo `docs/adr/`），并在相关 multica Issue 评论附 ADR 链接。
- 每周同步 Stage 进度与开放决策（O-1~O-7）状态；开放决策未关闭的 Story 不得进入 `in_progress`。
