# Story 3.2.1.2 — provider ctx 合规防护

> 状态：done ｜ 作为 SDK，我希望 provider（含 Jev HTTP）绝不能耗掉超过阶段预算的时间，以便补检/余量不被吃光。

## 前置依赖

- Story 3.2.1.1。

## 任务

#### T3.2.1.2.1 provider 调用包装器
- **目标：** 统一包装 provider 调用：注入阶段 ctx、透传取消、捕获超时映射。
- **产出：** `internal/deadline/guard.go`。
- **步骤：** 1) 包装函数确保传入 ctx 为阶段 ctx；2) ctx.Err()→ErrDeadlineExceeded 映射并标注阶段；3) 剩余预算计算暴露给 trace。
- **验收标准：** 包装层从不自建更宽 deadline；超时映射含阶段名。
- **依赖：** T3.2.1.1.2。
- **预估：** 0.5d

#### T3.2.1.2.2 阻塞 provider 传播 harness
- **目标：** 契约测试设施：注入"永不返回除非 ctx.Done"的 provider。
- **产出：** `internal/testutil/blockprov.go`。
- **步骤：** 1) select ctx.Done 的假 provider；2) 记录被取消时刻；3) 供 S4 契约套件与 S7 验收复用。
- **验收标准：** 阶段截止后 ≤50ms 内 provider 观察到取消（docs/05 §3 口径）。
- **依赖：** T3.2.1.2.1。
- **预估：** 0.5d

#### T3.2.1.2.3 root 越界守卫
- **目标：** 不启动任何会越过 root deadline 的阶段（docs/05 §2 provider 阻塞行）。
- **产出：** 编排器入口守卫（root 已到期 → 直接 ErrDeadlineExceeded，不调 provider）。
- **步骤：** 1) 每阶段启动前检查 root 剩余；2) 剩余 ≤0 时安全降级路径（后续 S5 接管动作）；3) 守卫测试。
- **验收标准：** root 过期后任何 provider 调用计数为 0。
- **依赖：** T3.2.1.2.1。
- **预估：** 0.5d

## Story DoD

- [ ] F-002 证据链完整：harness 用例 + 守卫用例进入 CI。

## 设计依据

docs/04 §1（"Jev/HTTP transport timeout 不得晚于阶段 deadline；provider 必须响应 ctx.Done()"）；docs/05 §2 矩阵。
