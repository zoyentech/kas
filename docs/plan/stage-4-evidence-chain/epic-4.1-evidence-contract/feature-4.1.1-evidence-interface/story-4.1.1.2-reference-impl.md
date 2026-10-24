# Story 4.1.1.2 — 参考实现与故障注入

> 状态：done ｜ 作为测试工程师，我希望有可控的 Evidence 源，以便构造有效期/权限/故障的全部边界场景。

## 前置依赖

- Story 4.1.1.1。

## 任务

#### T4.1.1.2.1 in-memory EvidenceProvider
- **目标：** 可编程返回指定 Evidence 集的参考实现。
- **产出：** `internal/evidence/inmem/provider.go`。
- **步骤：** 1) 按 Query 过滤（namespace/top_k/score 排序）；2) 记录调用与 ctx（供断言）；3) godoc 声明仅测试/示例用途，非生产。
- **验收标准：** 契约套件全绿；可精确控制返回集与顺序。
- **依赖：** T4.1.1.1.1。
- **预估：** 0.5d

#### T4.1.1.2.2 故障注入包装器
- **目标：** 对任意 provider 注入 timeout/error/慢响应/部分结果。
- **产出：** `internal/testutil/faultprov.go`。
- **步骤：** 1) 包装器模式（错误、延迟、挂起直到 ctx.Done、截断结果）；2) 用例演示；3) 供 S5 降级链与 S7 矩阵复用。
- **验收标准：** 每种故障可独立开关；与阻塞 harness（T3.2.1.2.2）互补。
- **依赖：** T4.1.1.2.1。
- **预估：** 1d

## Story DoD

- [x] 两套设施合入；S5/S7 任务可直接引用。

## 设计依据

docs/05 §2（故障场景矩阵）；docs/01 §2（最小运行依赖）。
