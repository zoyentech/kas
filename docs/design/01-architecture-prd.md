# KAS v1.2 软件架构与功能 PRD

**状态：** 开发前评审稿。**形态：** 内部 Go SDK；后续可由薄 HTTP/gRPC 服务包装相同核心接口。

## 1. 目标与边界

KAS 为宿主应用提供可测试、可观测的知识问答运行链，统一请求预算、Evidence 检索、知识有效期筛选、决策 provider、带引用回答、安全缓存和配置热替换。

### v1.2 目标

- 总请求预算拆分为初判、首轮、复判、补检和余量。
- 只使用当前授权且在有效期内的 Evidence；回答引用可追溯到最终 Evidence。
- 通过 DecisionProvider 隔离决策协议；Jev 可替换、可降级。
- 以 Health()、Prometheus 指标和 trace 版本字段支持运维与验收。
- Evidence 缓存按配置、知识代次和 principal scope 隔离。
- 配置加载、启动校验、构造候选运行快照后原子替换；在途请求固定旧快照。
- SDK 无会话持久状态，后续服务化无需重写领域逻辑。

### 不在 v1.2 范围

KAS 自有知识导入/切片/索引/管理 UI；独立服务进程；跨进程配置控制面；内置配置写入、History、Rollback API、CLI、文件 watcher；默认 Redis/分布式缓存；长期会话记忆。也不在未通过 GoldenSet 前替换用户可见链路。

## 2. 使用者与最小组件

| 使用者 | 目标 |
|---|---|
| Go 服务开发者 | 从 handler 调用 KAS，得到 answer、citations、trace/config 版本与稳定错误 |
| QA/产品 | 用冻结 GoldenSet 对比旧链和 KAS |
| 运维 | 查看 Health、超时、缓存和 fallback metrics |
| 配置维护者 | 经 Load/Validate/Reload 发布新快照 |

SDK 首版不新增 KAS 服务、数据库、消息队列或分布式缓存。最小运行依赖为宿主 Go 服务、一个 EvidenceProvider、宿主认证层提供的可信 principal scope，以及由宿主提供 scrape endpoint 的 Prometheus registry。Jev 可选；其不可用时只有在规则能安全判定时继续，否则必须 abstain/clarify。

## 3. 架构与请求流

~~~mermaid
flowchart LR
  A[宿主 API / 身份与授权] --> B[KAS SDK]
  C[配置源] --> D[Load + Validate]
  D --> E[不可变 Runtime Snapshot]
  E --> B
  B --> F[请求编排与逐阶段 deadline]
  F --> G[Evidence Cache]
  G -->|miss| H[EvidenceProvider]
  H --> I[权限与知识有效期筛选]
  G --> I
  I --> J[候选组]
  J --> K[DecisionProvider]
  K --> L[Jev Adapter]
  K --> M[确定性规则]
  K --> N[受约束决策缓存]
  I --> O[回答与引用校验]
  K --> O
  O --> P[Response]
  B --> Q[Health / Prometheus / Trace]
~~~

- **配置加载器/校验器**：校验预算等式、有效期策略、缓存上限和 provider 参数。
- **Runtime Manager**：构造不可变 snapshot，准备成功后原子替换。
- **编排器**：请求开始时读取一次 snapshot，按阶段预算执行检索、决策、补检和回答。
- **EvidenceProvider**：从宿主权威来源或检索服务返回统一 Evidence，不改变事实源所有权。
- **Evidence Cache**：缓存有效 Evidence，TTL 不超过知识有效期。
- **DecisionProvider**：将候选组映射为有限 action。
- **Jev Adapter**：封装 Jev 协议、响应校验、模型版本和降级。
- **回答组装器**：拒绝越权、过期或不在当前结果中的引用。
- **Health/metrics/trace**：输出依赖状态、低基数指标和版本诊断字段。

## 4. 端到端流程

1. 宿主完成认证授权，构造可信 PrincipalScope。
2. KAS 读取当前 Runtime Snapshot；把 config_version、provider/model/version 固定到该请求。
3. 根 deadline 为 min(caller deadline, now + total)。
4. 初判决定检索、澄清或 abstain。
5. 按当前配置和 scope 查 Evidence 缓存；hit 仍做当前授权与有效期核查，miss 调 EvidenceProvider。
6. 丢弃未经授权、尚未生效、已过期或有效期未知的 Evidence。
7. DecisionProvider 选择回答候选组、补检、澄清或 abstain。
8. 补检结果重新经过权限和有效期过滤。
9. 回答只引用最终有效 Evidence；仅缓存 Evidence，不缓存最终回答。
10. 写 trace/metrics，不记录原始问题、Evidence 正文或身份明文。

## 5. 功能需求

| ID | 优先级 | 要求 | 验收 |
|---|---|---|---|
| F-001 | P0 | 五阶段预算与 total 精确相等 | 启动和 Reload 拒绝不等式 |
| F-002 | P0 | 阶段 deadline 传播 | 不晚于 root/caller deadline |
| F-003 | P0 | 知识有效期 | expired/future/unknown 按策略过滤，不进答案或缓存 |
| F-004 | P0 | Evidence 安全缓存 | key 含规范化问题、config_version、principal scope |
| F-005 | P0 | Health() | required 故障 not_ready；optional Jev 故障 degraded |
| F-006 | P0 | Prometheus | 请求、阶段、deadline、缓存、有效期、provider、降级指标 |
| F-007 | P0 | DecisionProvider | 编排器不直接依赖 Jev 协议 |
| F-008 | P0 | Jev 降级链 | Jev → 规则 → 受约束决策缓存 → abstain/clarify |
| F-009 | P0 | 版本 trace | config_version、provider、model、model_version、fallback |
| F-010 | P0 | 安全 action 与引用 | action 属于枚举；目标组和 citation 来自当前候选 |
| F-011 | P0 | 原子 Reload | 失败不替换；在途请求不混用新快照 |
| F-012 | P0 | 无状态 SDK | 可随宿主副本扩展，本地 cache 只影响命中率 |
| F-013 | P1 | 共享 cache adapter | 后续可插入 Redis 等实现，保持安全 key |
| F-014 | P1 | 服务化包装 | 复用领域 API，不改问答语义 |

## 6. 配置、扩容与安全

config_version 基于规范化、可观测配置生成。secret 明文不参与计算；secret reference/revision 可参与。相同 snapshot 内请求、trace、响应元数据和 cache key 使用同一版本。

发布顺序：读取/解析 → Validate → 准备 provider/runtime → 生成版本 → 原子替换。失败时 current pointer 保持原值。新请求读新 snapshot；在途请求保留旧 snapshot。宿主可用自己的 watcher 或集中控制面调用 Reload，KAS 不包含这些控制面功能。

SDK 在请求间无会话持久状态，宿主可水平扩容。首版进程内 cache 不跨副本共享，命中率会分片但正确性不受影响。未来服务化可增加统一认证、限流、共享 cache 和配置下发；共享 cache 必须保持相同 scope、content generation、config version、TTL 与权限复核规则。

principal scope 由宿主可信认证/授权层生成。原始用户 ID、token、question、Evidence 正文不得作为 cache key、metric label 或常规 trace 属性。缓存命中仍验证当前权限；权限变化需改变 scope fingerprint 或 authorization epoch。Evidence 不完整或过期时只能澄清/拒答。

## 7. 首发门禁

GoldenSet 达到 [验收门槛](05-goldenset-acceptance.md)；过期 Evidence 使用为 0、跨 scope cache hit 为 0；reload 并发时配置版本不串线；Health/Prometheus/trace 可定位 fallback 和 timeout；敏感数据不出现在指标和常规日志。
