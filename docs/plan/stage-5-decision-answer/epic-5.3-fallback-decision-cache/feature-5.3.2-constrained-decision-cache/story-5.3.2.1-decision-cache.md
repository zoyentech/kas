# Story 5.3.2.1 — 决策缓存实现

> 状态：todo ǀ 作为降级链设计者，我希望决策缓存可复用但受严格约束，以便 Jev 抖动时减少 abstain 而不引入错误决策。

## 前置依赖

- T5.1.1.1.2（校验器）；S3 事件总线。

## 任务

#### T5.3.2.1.1 key 组成与存取
- **目标：** key = H(config_version, model/model_version, policy 版本, 候选组 fingerprint, corpus generation, scope fingerprint)。
- **产出：** `internal/decision/dcache/cache.go`。
- **步骤：** 1) 候选 fingerprint = 组 ID 集+证据版本摘要的规范化哈希；2) LRU+TTL（max_entries/ttl 配置）；3) 任一维度变化 miss 用例。
- **验收标准：** 维度隔离表驱动（六维）通过。
- **依赖：** 无。
- **预估：** 1d

#### T5.3.2.1.2 命中再校验
- **目标：** 命中后用当前输入重过 ValidateDecision；不通过 → miss 并继续链。
- **产出：** Get 包装 + 用例。
- **步骤：** 1) 复用统一校验器；2) 校验失败淘汰条目；3) 输出 Decision 标注 Provider=decision_cache、Degraded=true。
- **验收标准：** docs/05 §2 命中行自动化；伪造越界目标的缓存条目被拒。
- **依赖：** T5.3.2.1.1。
- **预估：** 0.5d

#### T5.3.2.1.3 接入降级链与禁写规则
- **目标：** 作为链第三级；abstain/clarify 终态结果不入缓存（只缓存可复用的正向决策，语义注明）。
- **产出：** 链注册 + 禁写规则 + 事件。
- **步骤：** 1) 实现 DecisionProvider 包装；2) 禁写：终态/低置信/合同错误结果；3) hit/miss 事件（S6 复用 cache_result 语义或独立标签，注明）。
- **验收标准：** 三级链端到端（Jev 失败→规则未命中→缓存命中）用例通过。
- **依赖：** T5.3.2.1.2；T5.3.1.1.1。
- **预估：** 1d

## Story DoD

- [ ] 决策缓存合入降级链；文档注明与 Evidence 缓存互不影响。

## 设计依据

docs/04 §3（决策缓存约束）；docs/02 decision.fallback.cache。
