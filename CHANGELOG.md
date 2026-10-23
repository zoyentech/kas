# Changelog

本项目遵循 [Semantic Versioning](https://semver.org/)；格式参考
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

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

### chore

- 开源治理基线：Apache-2.0、CONTRIBUTING、SECURITY、CODEOWNERS、模板、
  ADR-0001/0002、依赖 license 基线（O-1 组织/仓库名待确认，模块路径为
  占位）
- CI：Go 1.25/1.26 × linux/darwin 矩阵、lint+fmt 门禁、覆盖率阈值
  （60% 起）、acceptance job
