# KAS v1.1 → v1.2 迁移指南

> 本文件是 docs/design/06 的定稿版（Story 7.2.2.1 T2）；正文与设计文件
> 保持一致，落地补充见文末"实现对照"。


## 1. 原则

保留宿主现有用户可见链路和知识事实源，先接入 KAS adapter 并 shadow run；通过 GoldenSet 后内部试用，再小流量切换。KAS 是 query runtime，不接管内容事实源、导入或索引所有权。

如果宿主为 ai-support-platform，应先复核当前生产 answered path：已有历史设计将 PostgreSQL script_* 作为话术事实源，RAGFlow 是异步管理镜像；不能仅凭管理集成推定 RAGFlow 处于生产问答检索路径。发布前按当前代码、配置和真实请求重新确认。

## 2. 差异表

| 领域 | 可读的 v1.1 基线 | v1.2 |
|---|---|---|
| 交付 | 内部 Go SDK，后续服务化 | 保留，服务仅包装 SDK |
| 配置 | 讨论读写和动态热更新 | Load + Validate + 进程内 Reload 原子替换；无 History/Rollback/CLI/watcher |
| 超时 | 请求时限 | total 精确拆五阶段，逐级 context deadline，启动断言 |
| 决策 | Jev 直接用法待定 | DecisionProvider 抽象，Jev 为实现之一 |
| Jev 故障 | 降级链待定 | Jev → 规则 → 受约束决策缓存 → abstain/clarify |
| 缓存 | 无固定 contract | Evidence cache P0：规范化问题/config_version/principal scope，加权限 epoch/知识代次 |
| 知识时效 | provider 侧处理为主 | P0：过滤 expired/future/unknown，cache TTL 受有效期约束 |
| 运维 | 稳定性待验证 | P0：Health、Prometheus、带版本 trace |
| 扩容 | 先 SDK，后服务 | stateless SDK 可随宿主扩展；本地 cache 只影响 hit rate |

“可读的 v1.1 基线”只指引用对话可读到的目标和片段，不是对不可见 v1.1 正文的逐段 diff。

## 3. 配置迁移

1. 将旧 timeout 映射到 total 与五段预算，评审精确等式；校验失败即 fail-fast，不自动改写。
2. 将 provider/model、策略、Evidence cache、知识有效期和 Health 配置迁至 v1.2 schema。
3. secret 值改由宿主 secret manager/env 注入，YAML 只放引用。
4. 移除 KAS 自有配置写入、History/Rollback API、CLI 和 watcher 设想；宿主原有配置发布方式可调用统一 Reload。
5. config_version 只由脱敏规范化配置和 secret revision 生成，不包含 secret 明文。

## 4. 代码迁移

- 将 orchestration 对 Jev 的直连改为 DecisionProvider；Jev 协议、timeout、响应校验封装在 adapter。
- 统一候选组和有限 action enum；禁止自由文本决策直通执行。
- 将检索适配成 EvidenceProvider，补齐来源 ID、content version、permissions、ValidFrom/ValidUntil。
- 先启用 bounded in-memory Evidence cache；测试 key 隔离、TTL、validity 和 cache hit 权限复核。
- 接入 Health(ctx)、Prometheus registerer、TraceSink；/metrics 由宿主提供。
- Response 增加 trace/config version、degraded 标志和安全 reason code，不暴露上游错误文本。

## 5. 热替换语义

若 v1.1 有运行时更新入口，迁移为统一调用 Reload。更新顺序必须是 load/parse → validate → 准备依赖 → 原子替换。

- 在途请求固定旧 snapshot；新请求读取新 snapshot。
- 验证/依赖构建失败时 current pointer 不变。
- 配置回退由宿主配置系统重新发布上一份批准配置；SDK 不保存 history，也不提供 rollback command。
- config_version 变化自然隔离 cache；不要跨配置版本读取旧 cache entry。

## 6. 切换阶段

### A. Shadow run
仅运行脱敏 GoldenSet 或获批抽样；旧链继续返回结果。对比 Evidence、action、回答事实、引用、时延、成本。影子结果不展示给用户、不作为反馈回灌。

### B. 内部 QA
验证 Health、metrics、Jev fallback、cache scope、知识有效期和 Reload。不要通过关闭有效期校验或扩大总 deadline 来绕过失败。

### C. 小流量灰度
达到 GoldenSet 门槛且 P0 安全错误为零后小比例切流。监控 p95/p99、deadline、fallback、cache 隔离、过期 Evidence 和 provider errors。出现 P0 问题立即停止扩大，由宿主切回原链路。

### D. 扩展调用方
逐调用方迁移并保持领域契约。所有目标场景通过 GoldenSet 后再决定是否退役旧运行链。RAGFlow 是否保留应分别评估导入、管理、检索、事实源职责。

## 7. 回退边界

SDK 不实现配置 history/rollback。应用部署可以回到上一版本，宿主配置系统可以重新发布批准配置。回退时隔离新版本 cache namespace，避免旧配置复用新 version 的 Evidence/decision。不要以复用未验证的跨版本 cache entry 恢复结果。


---

## 实现对照（v1.2.0 落地状态）

| 迁移项 | 设计章节 | 实现落点 |
|---|---|---|
| 配置 Load/Validate/Reload 原子替换 | §3 | `kas.LoadConfig/LoadConfigFile/ValidateConfig`、`Client.Reload`（internal/runtime） |
| secret 仅 env 引用 | §3 | internal/decision/jev（env 注入，错误仅含 env 名） |
| orchestration → DecisionProvider | §4 | internal/decision（契约/规则/Jev/降级链）；宿主注入 provider 作为链首选级 |
| 检索适配 EvidenceProvider | §4 | internal/evidence（缓存化 provider + 有效期/权限过滤） |
| bounded in-memory cache | §4 | internal/cache（MemCache；key 隔离/TTL/权限复核测试齐备） |
| Health/Prometheus/TraceSink | §4 | internal/obs（health/metrics/trace；Registerer 注入） |
| Response 版本/degraded/reason | §4 | `kas.Response`（ConfigVersion/Degraded/Degradation 受控枚举） |
| 热替换语义 | §5 | 五步发布顺序；在途持旧快照（验收矩阵行 8/9） |
| 切换阶段（shadow→灰度） | §6 | 宿主侧推进；GoldenSet runner 与 19 行矩阵已就绪 |
| 回退边界 | §7 | SDK 无 history/rollback；cache 按 config_version 自然隔离 |
