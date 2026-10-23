# Story 5.2.1.1 — Jev 协议封装

> 状态：todo ǀ 作为 SDK，我希望 Jev 协议被完全封装，以便协议演进只改一个包。

## 前置依赖

- Epic 5.1；O-2（协议文档或 fake 先行）。

## 任务

#### T5.2.1.1.1 env 注入与客户端配置
- **目标：** endpoint/api_key 从环境变量读取（名称来自配置）；缺失时的行为定义（构造期报错或 Health degraded，按 docs/05 矩阵选构造期报错并注释）。
- **产出：** `internal/decision/jev/config.go`。
- **步骤：** 1) 读取 env；2) 值仅存内存，Stringer/redaction 防打印；3) 缺失/为空报 ErrInvalidConfig（字段路径指 env 名）。
- **验收标准：** 错误与日志中 api_key 值零出现（扫描测试）。
- **依赖：** O-2。
- **预估：** 0.5d

#### T5.2.1.1.2 请求/响应映射与 fake server
- **目标：** DecisionInput ↔ Jev 协议映射；fake Jev server 支持脚本化响应。
- **产出：** `internal/decision/jev/client.go`；`internal/testutil/jevfakeserver.go`。
- **步骤：** 1) 按 O-2 文档映射（无文档期按 docs/03 §3 最小集）；2) fake server 可配置延迟/状态码/响应体；3) 映射单测。
- **验收标准：** 请求包含候选组必要字段；fake server 覆盖 200/timeout/5xx/畸形体。
- **依赖：** T5.2.1.1.1。
- **预估：** 1.5d

#### T5.2.1.1.3 单次尝试与传输超时
- **目标：** max_attempts=1 执行策略；transport timeout = min(阶段剩余, 配置上限)。
- **产出：** `internal/decision/jev/call.go`。
- **步骤：** 1) 拒绝重试（即使配置>1 也按 docs/03 §3 语义单次，配置校验已限制）；2) http.Client 超时随阶段 ctx；3) 超时→ErrDeadlineExceeded 映射。
- **验收标准：** 慢响应在阶段截止被切断（阻塞 harness 复用）；无第二次调用（计数断言）。
- **依赖：** T5.2.1.1.2。
- **预估：** 1d

## Story DoD

- [ ] adapter 可作为 DecisionProvider 被编排器调用（fake server 驱动）。

## 设计依据

docs/03 §3（Jev 约束）；docs/02 decision.jev；docs/04 §1（transport 约束）。
