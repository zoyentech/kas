# Story 3.3.1.1 — Answer 主流程骨架

> 状态：done ｜ 作为宿主开发者，我希望 Client.Anway 有稳定主流程，以便 S4/S5 能力渐进注入而不破坏结构。

## 前置依赖

- Epic 3.2（deadline 引擎）；S2（snapshot）。

## 任务

#### T3.3.1.1.1 主流程与快照固定
- **目标：** `Client.Answer(ctx, req)`：校验 Request→读 snapshot→固定 config_version→进入阶段流水线→组装 Response。
- **产出：** `internal/orchestrator/answer.go`。
- **步骤：** 1) Request 校验（复用 T3.1.1.1.1）；2) snapshot.Load() 一次并传值；3) 流水线调用与 Response 骨架（TraceID 生成挂 S6）。
- **验收标准：** 单请求全程 config_version 不变；非法 Request 返回 ErrInvalidRequest。
- **依赖：** 无。
- **预估：** 1d

#### T3.3.1.1.2 初判阶段挂钩
- **目标：** preliminary 接口（占位实现：默认放行检索路径）。
- **产出：** `internal/orchestrator/preliminary.go` + 接口定义。
- **步骤：** 1) 接口：输入 Request+快照，输出（continue/clarify/abstain + reason）；2) 默认实现 continue；3) S5 规则引擎替换点注释。
- **验收标准：** 阶段预算内执行；三种输出的流向分支有测试。
- **依赖：** T3.3.1.1.1。
- **预估：** 0.5d

#### T3.3.1.1.3 补检挂载点与阶段串联
- **目标：** first_round/rejudge/supplement/reserve 挂载点定义与五阶段串联执行器。
- **产出：** `internal/orchestrator/pipeline.go`。
- **步骤：** 1) 每阶段接口（ctx、snapshot、阶段中间状态 in/out）；2) 串联执行器按 docs/04 §1 派生各阶段 ctx；3) supplement 结果"重新过滤"的钩子留给 S4（注释标记 docs/01 §4 第 8 步）。
- **验收标准：** fake 阶段注入下五阶段按序执行、每阶段拿到独立派生 ctx、root 守卫生效。
- **依赖：** T3.3.1.1.2。
- **预估：** 1d

## Story DoD

- [x] 骨架端到端测试（fake 阶段）通过；S4/S5 挂载点评审冻结。

## 设计依据

docs/01 §3~§4；docs/04 §1。
