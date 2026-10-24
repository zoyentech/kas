# Story 7.1.2.2 — 降级场景套件

> 状态：done ǀ 作为值班运维，我希望全部故障形态的行为被场景固化，以便升级不引入回归。

## 前置依赖

- Epic 5.2/5.3（fake Jev server、降级链）。

## 任务

#### T7.1.2.2.1 Jev 故障族
- **目标：** timeout/5xx/非法结构（非法 action/越界 ID/NaN）三形态：按序降级、contract error 记录、不超 root deadline。
- **产出：** `acceptance/matrix/jev_test.go`。
- **步骤：** 1) fake server 三形态驱动；2) 断言降级路径（route 顺序）与 Response.Degradation；3) 断言全程 ≤ root deadline（50ms 口径）。
- **验收标准：** docs/05 §2 三行（Jev timeout/5xx、非法响应）全绿。
- **依赖：** 无。
- **预估：** 1d

#### T7.1.2.2.2 规则冲突与决策缓存命中
- **目标：** 多规则冲突优先级稳定可解释；决策缓存命中时候选/版本/scope 匹配且 action 重校验。
- **产出：** `acceptance/matrix/rules_cache_test.go`。
- **步骤：** 1) 冲突规则集用例（断言命中名与顺序）；2) 缓存命中/伪造条目拒用两用例；3) trace 可解释断言（reason 含规则名）。
- **验收标准：** docs/05 §2 两行（规则冲突/决策缓存命中）全绿。
- **依赖：** T7.1.2.2.1。
- **预估：** 0.5d

#### T7.1.2.2.3 provider 阻塞与 deadline 场景
- **目标：** provider 挂起：ctx 取消传播、不启动越过 root deadline 的阶段；空结果/低相关 → clarify/abstain 不编造。
- **产出：** `acceptance/matrix/deadline_test.go`。
- **步骤：** 1) 阻塞 harness 全阶段注入；2) provider 调用计数断言（root 过期后零调用）；3) no-answer 语料断言动作。
- **验收标准：** docs/05 §2 两行（provider 阻塞、空结果/低相关）全绿。
- **依赖：** T7.1.2.2.1。
- **预估：** 1d

## Story DoD

- [x] 降级/阻塞 5 行矩阵全绿。

## 设计依据

docs/05 §2 降级与阻塞行、§3 deadline 口径。
