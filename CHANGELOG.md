# Changelog

本项目遵循 [Semantic Versioning](https://semver.org/)；格式参考
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [v1.4.2] - 2026-10-27

### fix(example)

- 演示余额改为时间锚定 sawtooth 趋势函数：修复重启重置导致的看板余额回跳与
  扣费过快穿底（KAS 余额链路本身无问题，均为示例缺陷；commit 6ffcd44）
- 新增真实凭据模式 `KAS_DEMO_FAKE=0`：注入 DEEPSEEK_TOKEN / ALIBABA_CLOUD AK·SK /
  QWEN_APIKEY 后 deepseek→官方 /user/balance、aliyun/qwen→BSS
  QueryAccountBalance（Jev 无官方接口保持 fake，O-2）；实测与控制台一致
- monitoring.md 补真实/模拟切换说明

### chore

- 移除误提交的 16MB `metrics` 编译产物并加入 .gitignore（根目录 go build 裸名输出）
- 库代码（根包/internal）相对 v1.4.1 零变更——本版为示例/文档/仓库卫生版本，
  对 `go get` 使用者无行为影响

## [v1.4.1] - 2026-10-27

### feat(balance)

- 新增内置 `qwen` 余额源（组件视图）：`QWEN_APIKEY` 存在且阿里云 BSS 凭据可用时
  自动注册；金额=阿里云账户现金余额（DashScope/Qwen 模型计费口径，与 `aliyun`
  源同源），阈值/告警独立（`balance.sources.qwen`）；看板按源分列自动展示

## [v1.4.0] - 2026-10-27

多源余额监测与运行时热更新（F-016 扩展；参考 ai-support-platform 管理监控页 #1243）。

### feat(balance)

- **多源余额**：BalanceSource 升级为命名源注册表（balance.Manager）——每源独立
  轮询/阈值/周期；`kas_balance_amount{source,unit}` / `kas_balance_state{source}`
  按源分列；Health 信息组件改为 `balance:<source>`
- **内置适配器**（按 env 存在自动注册，零新增依赖）：
  - `deepseek`：官方 `GET /user/balance` 契约（is_available/balance_infos，多币种优先 CNY）
  - `aliyun`：阿里云 BSS QueryAccountBalance（账户现金余额=Qwen/百炼计费口径；
    POP RPC V1 HMAC-SHA1 签名自实现，需 RAM BSS 只读权限）
  - `jev`：v1.3 最小契约端点（O-2）；宿主注入源注册为 `host`
- **运行时热更新 API**（宿主管理后台接入点，无需重建 Client）：
  `UpsertBalanceSource` / `RemoveBalanceSource` / `SetBalanceConfig`；
  编程查询 `BalanceOf` / `BalanceSources`
- 配置：`balance.sources.<name>` 按源覆盖全局阈值/周期（校验合成后 critical < low）

### feat(monitoring)

- 看板多源分列：`balance_source` 模板变量 + "余额状态 · <源>"repeat 卡 +
  "组件余额总览"表；余额趋势按源多线（unit 入 legend）
- `examples/metrics` 三源演示（jev credit / deepseek CNY / aliyun CNY 递减）

### chore

- ADR-0003 v1.4 修订（多源与运行时变更 API 边界）；Stage 8 计划增 Epic 8.8；
  Health 余额组件键由 `balance` 改为 `balance:<source>`（行为变更，管理页按新键取用）

## [v1.3.0] - 2026-10-27

运营洞察与接入稳定化（展示层走 Prometheus+Grafana，ADR-0003；F-015~F-019）。

### feat(stability)

- provider panic 隔离（F-019）：deadline.Call / 降级链每级 / 阶段挂载点 / 组装器
  四层守卫——任意三方 provider panic 不再击穿宿主进程，归一为契约违规走降级，
  panic 值原文不进入任何输出面；`kas_provider_errors_total{error_class="panic"}` 计数
- Jev 熔断（F-019，默认开启）：连续失败 5 次进 open（快速失败直落降级链，
  不再烧 rejudge 超时预算）、冷却 30s 后 half-open 单探测、成功自动回闭；
  `kas_provider_breaker_state` gauge + Health detail 可见；配置
  `decision.jev.breaker.{enabled,failure_threshold,cooldown}`

### feat(balance)

- 余额监测（F-016）：BalanceSource 契约（宿主可注入任意计费源）+ Jev 适配
  （O-2 协议未定，最小契约 + fake 先行）+ 后台轮询（失败指数退避、unknown 保留
  上次成功值）+ 四态判定 normal/low/critical/unknown（阈值可配）；
  `kas_balance_amount{source,unit}` / `kas_balance_state{source}` gauge、
  BalanceEvent、`Client.Balance()`、Health balance 信息组件；
  配置节 `balance:` + `decision.jev.balance_endpoint_env`

### feat(monitoring)

- 监控栈交付（F-015/F-017/F-018）：`monitoring/` 一键 docker compose
  （Prometheus+Grafana 版本固定、datasource/看板自动 provisioning、30 天保留）
  + 预置 KAS Overview 看板（健康/成功率/QPS/时延分位/阶段/缓存/降级/错误/
  余额/熔断 14 面板）+ `examples/metrics` 宿主 /metrics 可运行演示
  （fake Jev、余额递减、持续流量）+ `docs/monitoring.md` 指南与 PromQL 速查
- 指标面新增 label：`source`/`unit`（余额 gauge，有界枚举）

### chore

- ADR-0003 accepted（含路线 A 修订）；Stage 8 计划按路线重组
  （8.1/8.3/8.4 deferred，新增 8.7 监控栈交付）；契约套件增 panic 用例

## [v1.2.0] - 2026-10-24

首个开源版本（MVP：P0 需求 F-001~F-012 全部落地；GoldenSet 验收通过建议门槛）。

### feat(config)

- YAML 严格解析（未知字段拒绝、duration 仅字符串形态）与全部启动校验
  （预算等式 ±1ns、溢出、provider 超时装入阶段预算、reserve 下限、
  缓存容量、有效期策略完整性、label allowlist、trace 正文 fail-closed）
  （F-001，T2.1.x）
- 脱敏 config_version（规范化确定性序列化 + SHA-256 前 12 hex；secret
  明文不参与、引用名参与）与原子 Reload（失败保持旧快照、在途请求持旧
  版本）（F-011，T2.2.x）

### feat(orchestrator)

- 五阶段编排（初判→首轮→复判→补检→余量）：每阶段仅从 root 派生
  deadline（编译期防上一阶段派生）、阶段结束即取消、root 越界守卫
  （F-002，T3.2.x/T3.3.x）
- 领域类型与七类哨兵错误（errors.Is 可判定）、公共错误面映射（上游
  文本零外泄）（T3.1.x）
- 内部事件总线（11 类事件，覆盖 docs/04 §5 全部指标与 §6 trace 字段）

### feat(evidence)

- 知识有效期过滤：expired/future/unknown fail-closed（配置必须显式
  fail_closed=true）、来源 max-age 推导截止供缓存 TTL 取 min（F-003）
- 权限复核：声明式 TagChecker + 宿主注入函数双模式（O-7）；缓存命中后
  强制复核、不足量回源、补检结果重新过滤
- Evidence 安全缓存：NFKC 规范化（否定词/数字/单位/型号符号保留属性
  测试）、七维 key（仅不可逆摘要存储）、三重上限确定性 LRU、
  TTL=min(配置,新鲜度,最早截止)、五类禁写（F-004）

### feat(decision)

- 统一响应校验（action 枚举/目标组归属/NaN-Inf-[0,1]）与契约测试套件
  （F-007/F-010）
- 确定性规则引擎（声明序首命中、双用途决策+初判、配置期拒绝未知条件）
- Jev adapter（O-2 协议文档未到位，按 docs/03 §3 最小契约 + fake server
  先行）：env 注入（api_key 零出现）、单次尝试、严格 JSON 解析、模型
  版本捕获
- 降级链 Jev→规则→决策缓存→abstain（顺序固定、每级错误分类续链、
  安全护栏可覆盖上游 answer、预算守卫）（F-008）
- 受约束决策缓存（六维 key、命中再校验、终态/低置信禁写）
- 回答组装与引用安全：citation ⊆ 最终有效 Evidence、保序截断
  max_citations、零引用 answer 终检拦截、无证据终态 output_on_unknown

### feat(observability)

- Health 聚合（required 故障 not_ready / optional degraded / 短独立超时
  并发探测）（F-005）
- Prometheus collector（docs/04 §5 全表；label allowlist 只能收窄；
  CheckLabelValues 禁入扫描；宿主 Registerer 注入、Close 注销）（F-006）
- TraceSink（include 白名单 + 敏感键硬拒、128bit trace_id 全事件一致、
  版本链 config_version/provider/model/model_version/fallback_route 完整）
  （F-009）

### test(acceptance)

- GoldenSet 框架：用例 schema/加载校验、102 条去标识化合成样本（七类
  场景、P0 子集 30 条内容 hash 冻结）、runner 与评分器（Recall@k/
  Precision@k/MRR/nDCG/事实支持/不可答准确率/时延分位，公式单测）
- docs/05 §2 验收矩阵 19 行场景套件（安全/降级/并发）全部自动化
- 发布审计：五输出面敏感值零泄漏扫描（指标/trace/错误/响应/health）

### fix(deps)

- golang.org/x/text 升级 v0.39.0（GO-2026-5970，go 基线随之 1.23→1.25，
  ADR-0002 修订）；govulncheck 影响代码漏洞 0（部署工具链建议 ≥ go1.26.10，
  覆盖 stdlib GO-2026-6809 等 14 项）

### chore

- 开源治理基线：Apache-2.0、CONTRIBUTING、SECURITY、CODEOWNERS、模板、
  ADR-0001/0002、依赖 license 基线（O-1 已确认：github.com/zoyentech/kas）
- CI：Go 1.25/1.26 × linux/darwin 矩阵、lint+fmt 门禁、覆盖率阈值
  （60% 起）、acceptance job
