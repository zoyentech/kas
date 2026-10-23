# Stage 2 — 配置与运行时快照

> 状态：todo ｜ 前置：S1 ｜ 里程碑：M2 ｜ 粗估：8 人日

## 目标

docs/02 示例 YAML 端到端可用：类型系统与严格加载、全部启动校验（含 F-001 预算等式）、脱敏 config_version、不可变 Runtime Snapshot 与原子 Reload（F-011）。失败配置绝不替换当前快照，在途请求固定旧快照。

## 范围内

`LoadConfig/LoadConfigFile/ValidateConfig` 公共 API；YAML 严格解析与 duration 校验；预算等式/溢出/上限/provider 参数/缓存容量校验；config_version 规范化哈希；snapshot 构建与 `atomic.Pointer` 替换；`Client.Reload` 与并发语义测试。

## 范围外

编排器如何消费快照（S3）；Evidence/Decision provider 实现（S4/S5）；Health/metrics/trace（S6）。

## 入口条件

- S1 出口条件全部满足。

## 出口条件（DoD）

- [ ] F-001：任何 total ≠ 五段之和（含溢出）的配置在 New/Reload 均被拒绝，单测覆盖。
- [ ] F-011：Reload 失败保持旧 config_version；并发 Answer 模拟下新旧快照不串线（`-race` 通过）。
- [ ] docs/02 示例 YAML 可被 LoadConfigFile 加载并通过 ValidateConfig。
- [ ] 错误信息含字段路径、不含 secret 与整份配置；config_version 不含 secret 明文。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 2.1 | 配置 schema 与校验 | 2.1.1 配置模型与 YAML 加载；2.1.2 预算等式与参数校验 |
| 2.2 | 版本与原子 Reload | 2.2.1 脱敏 config_version；2.2.2 原子 Reload 与 snapshot |

## 需求覆盖

F-001（2.1）、F-011（2.2）、F-009 部分（config_version 生成，2.2.1）。

## 风险与注意事项

- 严格解析（KnownFields）与 yaml.v3 默认行为差异需在测试中固化，防止静默吞掉未知字段。
- O-4/O-5（宿主上限、reserve 下限）影响 2.1.2.1 的两条校验，先按默认值实现并留配置开关。
