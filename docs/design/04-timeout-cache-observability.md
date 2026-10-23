# 超时预算、缓存与监控设计

## 1. 逐阶段 deadline

不变量：

~~~text
total = preliminary + first_round + rejudge + supplement + reserve
~~~

参考 4800ms 配置：

| 阶段 | 时间 | 用途 |
|---|---:|---|
| 初判 | 400ms | 输入范围、澄清与检索路径 |
| 首轮 | 1800ms | cache/检索、候选组 |
| 复判 | 900ms | 对首轮有效候选做决策 |
| 补检 | 1100ms | 额外检索与最终校验 |
| 余量 | 600ms | 引用检查、组装、编码 |
| 合计 | 4800ms | 精确等于 total |

这是默认样例，不是性能承诺；应由 GoldenSet 和线上容量数据校准。

### context 规则

- root deadline = min(caller deadline, start + total)。
- 阶段 deadline = min(stage start + stage budget, root deadline)。
- 每个阶段从 root context 派生 context.WithDeadline；阶段结束取消，再从 root 派生下一阶段。
- 不从上一阶段的短 deadline context 派生下一阶段，否则子 context 无法延长父 deadline，会意外把首阶段预算当成总预算。
- caller deadline 更早时不可延长。Jev/HTTP transport timeout 不得晚于阶段 deadline；provider 必须响应 ctx.Done()。
- 可选阶段未运行时不做预算重分配，根 deadline 仍为上限。

~~~go
root, cancelRoot := context.WithDeadline(parent, minDeadline(parent, startedAt.Add(cfg.Timeouts.Total)))
defer cancelRoot()

preCtx, cancelPre := context.WithDeadline(root, minDeadline(root, preStart.Add(cfg.Timeouts.Preliminary)))
result, err := preliminary.Decide(preCtx, input)
cancelPre()

firstCtx, cancelFirst := context.WithDeadline(root, minDeadline(root, firstStart.Add(cfg.Timeouts.FirstRound)))
evidence, err := evidenceProvider.Search(firstCtx, query)
cancelFirst()
~~~

所有路径均调用 cancel。超时后可进入安全降级，但不得超过 root deadline。

### 启动校验

拒绝：预算为非正数；duration 加法溢出；total 与五段之和不等；total 超宿主上限；provider timeout 超阶段预算；reserve 小于经评审的最小组装预算；重试/backoff 无法装入阶段预算；未知配置字段或 duration 格式错误。错误指出字段路径，不打印整份配置或 secret。

## 2. Evidence 缓存

### Key 维度

~~~text
H(normalize(question), config_version, principal_scope_fingerprint,
  authorization_epoch, knowledge_namespace, corpus_generation,
  retrieval_policy_version)
~~~

核心字段至少有规范化问题、config_version、principal scope；再加知识空间、内容代次和授权 epoch 防止策略/内容变化后复用旧条目。存储仅使用不可逆 key 摘要，不保留原始问题和身份明文。

规范化使用 Unicode NFKC、首尾 trim、连续空白折叠。只对明确不区分大小写的字段做大小写归一；保留否定词、数字、单位、型号/药品符号和语义标点。

### 缓存规则

- 仅缓存 Evidence 列表与来源版本/有效期/授权元数据，不缓存最终回答。
- 空结果、provider error、timeout、权限不一致、有效期未知的结果不缓存。
- 读 cache 后仍用当前权限和当前时刻复核；过滤 expired/future Evidence 后不足量时可回源。
- TTL = min(configured TTL, source freshness TTL, earliest ValidUntil - now)；小于等于零不写入。
- 限制 max entries、总字节和单项字节；超限确定性淘汰并计数。
- 更改 config、授权 epoch 或 corpus generation 自然更换 key；按 namespace 可主动清理。
- v1.2 默认进程内 cache。多副本命中率会不同但正确性不变；共享 cache 是后续 adapter，不是首版依赖。

## 3. Jev 降级

顺序固定并写 trace：Jev 一次尝试 → 确定性规则 → 受约束决策缓存 → abstain/clarify。

规则必须可审计，条件完整，输出有限 action。决策缓存 key 绑定 config/model/policy 版本、候选 fingerprint、知识代次和 principal scope；命中后再次校验 action 和目标组。任何降级都设置 degraded=true 并增加 reason/route 指标。上游错误、模型思维链和 prompt 不给用户看。

## 4. Health

- not_ready：配置未加载/无效、必需 EvidenceProvider 不可用。
- degraded：可选 Jev 不可用，但规则链安全可用。
- healthy：所有必需组件正常。

Health 使用短独立 check timeout，不调用完整问答、不访问用户 Evidence cache、不运行模型生成。昂贵远端依赖由宿主控制探测频率。

## 5. Prometheus

SDK 接收宿主 prometheus.Registerer 并注册 collector，不自己创建 /metrics 服务。

| 指标 | 类型 | 有界标签 |
|---|---|---|
| kas_requests_total | Counter | operation, outcome, action |
| kas_request_duration_seconds | Histogram | operation, outcome |
| kas_stage_duration_seconds | Histogram | stage, provider, outcome |
| kas_stage_deadline_exceeded_total | Counter | stage |
| kas_deadline_exceeded_total | Counter | deadline_stage |
| kas_evidence_cache_total | Counter | result=hit/miss/bypass/error |
| kas_evidence_cache_evictions_total | Counter | reason |
| kas_evidence_filtered_total | Counter | reason=expired/future/unauthorized/unknown_validity |
| kas_provider_errors_total | Counter | provider, operation, error_class |
| kas_decision_fallback_total | Counter | route, reason |
| kas_health_status | Gauge | component, status |

禁止将 question、principal/user/tenant、Evidence ID、trace ID、错误原文、config_version 或任意自由字符串作为 label。模型版本默认只写 trace，避免高基数。

## 6. Trace 字段

记录 trace_id、config_version、policy version、Evidence/Decision provider、model、model_version、cache hit/miss、fallback route、阶段耗时和剩余 deadline、候选数量、action、过滤数量和 reason code。不记录密钥、question 正文、prompt、Evidence 正文或身份明文；诊断 query 如获批准也只用不可逆 hash/受控采样。
