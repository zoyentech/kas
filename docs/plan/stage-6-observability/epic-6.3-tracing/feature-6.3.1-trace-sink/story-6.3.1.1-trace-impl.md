# Story 6.3.1.1 — trace 实现

> 状态：done ǀ 作为排障工程师，我希望 trace 能定位每次 fallback 与 timeout 的版本链路，且绝不携带敏感内容。

## 前置依赖

- S3 事件总线与 TraceSink 接口。

## 任务

#### T6.3.1.1.1 TraceEvent 与默认 sink
- **目标：** 事件→TraceEvent 映射；默认结构化 sink（可注入自定义）。
- **产出：** `internal/obs/trace/sink.go`。
- **步骤：** 1) 字段结构对齐 docs/04 §6；2) no-op/结构化两种内置；3) 注入点（Dependencies.TraceSink / Option）。
- **验收标准：** 全链 fake 请求产生完整 trace 序列（阶段/决策/缓存/降级）。
- **依赖：** 无。
- **预估：** 0.5d

#### T6.3.1.1.2 include 白名单与脱敏器
- **目标：** 字段级白名单过滤器 + 敏感值拦截。
- **产出：** `internal/obs/trace/redact.go`。
- **步骤：** 1) include 配置驱动输出字段；2) 脱敏器：未知字段丢弃、敏感键名（question/prompt/evidence/tenant/token）硬拒；3) 扫描测试。
- **验收标准：** 注入敏感字段的场景输出为零泄漏（子串级断言）。
- **依赖：** T6.3.1.1.1。
- **预估：** 1d

#### T6.3.1.1.3 TraceID 生成与透传
- **目标：** 每请求生成 TraceID；Response.TraceID 透传；事件全程携带。
- **产出：** ID 生成（随机 128bit hex）+ 接线。
- **步骤：** 1) 生成器；2) Request.RequestID 存在时关联（不覆盖语义注明）；3) 断言全事件同 ID。
- **验收标准：** 同请求全部 trace 记录 trace_id 一致；Response 正确返回。
- **依赖：** T6.3.1.1.1。
- **预估：** 0.5d

#### T6.3.1.1.4 版本字段完整性
- **目标：** config_version、provider/model/model_version、fallback route、cache 结果在 trace 完整出现（F-009 验收）。
- **产出：** 完整性断言测试（覆盖 Jev 正常与降级两条路径）。
- **步骤：** 1) 两条路径各跑一请求；2) 断言版本链字段齐备；3) 与 docs/05 §2"可由 trace 定位 fallback/timeout"对应。
- **验收标准：** 字段清单核对留档；断言进 CI。
- **依赖：** T6.3.1.1.2；S5（Jev 路径可用）。
- **预估：** 1d

## Story DoD

- [x] F-009 证据留档；脱敏与完整性测试进入 CI。

## 设计依据

docs/04 §6；docs/02 tracing 节；docs/03 §6（TraceSink）。
