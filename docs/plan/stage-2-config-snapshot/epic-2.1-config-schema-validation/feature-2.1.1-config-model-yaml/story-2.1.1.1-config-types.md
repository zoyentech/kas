# Story 2.1.1.1 — 配置类型定义

> 状态：done ｜ 作为 SDK 集成方，我希望 YAML 中每个字段都有强类型与文档对应，以便配置即契约。

## 前置依赖

- S1（module 骨架）。

## 任务

#### T2.1.1.1.1 定义全部配置 Go 类型
- **目标：** 按 docs/02 全节定义类型与 yaml 标签。
- **产出：** `internal/config/types.go`。
- **步骤：** 1) 覆盖 schema_version/timeouts/evidence/decision/answer/health/metrics/tracing 全部字段；2) 每字段 godoc 注释对应 docs/02 行；3) `internal/config/example_test.go` 固化字段映射。
- **验收标准：** docs/02 YAML 非严格解析后与预期结构逐字段相等（表驱动断言）。
- **依赖：** 无。
- **预估：** 1d

#### T2.1.1.1.2 严格解析（未知字段拒绝）
- **目标：** KnownFields 严格解码，未知字段报错。
- **产出：** `internal/config/decode.go`。
- **步骤：** 1) yaml.v3 Decoder + KnownFields(true)；2) 错误包装附行号/字段路径。
- **验收标准：** 注入未知字段的 YAML 被拒且错误定位到字段名；合法文件零误报。
- **依赖：** T2.1.1.1.1。
- **预估：** 0.5d

#### T2.1.1.1.3 duration 解析与错误路径
- **目标：** 全部 duration 字段按 Go time.Duration 字符串校验（如 `4800ms`、`720h`）。
- **产出：** `internal/config/duration.go` + 表驱动测试（含 `1.5h`、`90s`、`0`、负数、非法串）。
- **步骤：** 1) `time.ParseDuration` 封装；2) 非法值错误含字段路径；3) 测试覆盖 docs/02 出现的全部 duration 形态。
- **验收标准：** 非法 duration 的错误信息含字段路径与原值；合法形态全部通过。
- **依赖：** T2.1.1.1.1。
- **预估：** 0.5d

## Story DoD

- [ ] 类型/严格解析/duration 三项合入，覆盖率 ≥ 90%（P0 安全路径）。

## 设计依据

docs/02（schema）；docs/03 §1（Config/TimeoutBudget 形态）；docs/04 §1（"未知配置字段或 duration 格式错误"拒绝项）。
