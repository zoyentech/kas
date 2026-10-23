# Feature 2.1.1 — 配置模型与 YAML 加载

> 状态：todo ｜ Epic 2.1 ｜ 粗估：2 人日

## 目标

Config 全量 Go 类型 + YAML 严格加载：未知字段拒绝、duration 按 Go time.Duration 格式解析、错误带字段路径。

## 设计要点

- 类型清单（docs/03 §1 + docs/02 全节）：Config、TimeoutBudget、EvidenceConfig（provider/namespace/top_k/max_candidate_groups/validity/cache）、DecisionConfig（jev/fallback order/rules/decision cache）、AnswerConfig、HealthConfig、MetricsConfig、TracingConfig。
- `endpoint_env`/`api_key_env` 只存环境变量名，绝不存值（docs/03 §1 密钥约束）。
- yaml.v3 `KnownFields`（严格模式）+ duration 字符串校验（docs/02 首行注释）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 2.1.1.1 | 配置类型定义 | 3 |
| 2.1.1.2 | Load/Validate API | 3 |

## 验收标准

- docs/02 逐字段可加载；未知字段、错误 duration 格式、类型不匹配均报错且路径准确。
- 类型上有 godoc，标注与 docs/02 字段的对应关系。

## 需求关联

F-001 前置；工程基线。
