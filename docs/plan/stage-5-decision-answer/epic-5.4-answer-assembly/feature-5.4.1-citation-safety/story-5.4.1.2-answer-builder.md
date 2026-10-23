# Story 5.4.1.2 — 回答组装器

> 状态：done ǀ 作为宿主开发者，我希望 Response 组装在 reserve 阶段一次完成，以便预算内产出完整可追溯响应。

## 前置依赖

- Story 5.4.1.1；S3 流水线。

## 任务

#### T5.4.1.2.1 组装器主体
- **目标：** reserve 阶段：目标组+已验证引用+Decision 元数据 → Response（Answer/Citations/Confidence/TraceID/ConfigVersion/Degraded）。
- **产出：** `internal/orchestrator/builder.go`。
- **步骤：** 1) 组装函数（无 IO，纯转换）；2) Confidence 透传约束（nil 或 [0,1]）；3) TraceID/ConfigVersion 由快照与请求上下文注入。
- **验收标准：** 字段完备性测试；组装耗时计入 reserve 预算（阶段内执行断言）。
- **依赖：** 无。
- **预估：** 1d

#### T5.4.1.2.2 至少一条有效引用强制
- **目标：** answer 动作输出必须含 ≥1 条已验证引用（纯对话模式未开启时）。
- **产出：** 终检断言 + 用例。
- **步骤：** 1) 终检失败 → 降级 abstain 并计 reason（citation_missing）；2) 该断言为最后一道闸（docs/03 §2）；3) P0 断言"无据回答=0"用例。
- **验收标准：** 构造零引用 answer 路径被终检拦截。
- **依赖：** T5.4.1.2.1。
- **预估：** 0.5d

## Story DoD

- [ ] 组装器合入 reserve 阶段；端到端（S3+S4+S5 全链 fake）测试通过。

## 设计依据

docs/03 §2（answer 至少一条有效引用）；docs/04 §1（reserve 用途）。
