# Story 3.2.1.1 — root/阶段 deadline 计算

> 状态：todo ｜ 作为编排器，我希望时间预算的推导集中且无捷径，以便任何阶段都无法突破 root deadline。

## 前置依赖

- S2（TimeoutBudget 已验证）。

## 任务

#### T3.2.1.1.1 minDeadline 与 root 构造
- **目标：** `root = min(caller deadline, start+total)`；caller 无 deadline 时取 start+total。
- **产出：** `internal/deadline/deadline.go`。
- **步骤：** 1) 读取 parent Deadline；2) min 计算（相等/更早/无 deadline 三态）；3) `WithDeadlineCause` 注入阶段化原因。
- **验收标准：** caller 更早不可延长（docs/04 §1 硬规则）三态用例齐备。
- **依赖：** 无。
- **预估：** 0.5d

#### T3.2.1.1.2 阶段派生工厂（只从 root 派生）
- **目标：** `Stage(root, name, budget)` 返回派生 ctx+cancel；签名不含上一阶段 ctx。
- **产出：** `internal/deadline/stage.go`。
- **步骤：** 1) `min(root deadline, stageStart+budget)`；2) 工厂签名强制 root 参数（编译期防护）；3) 运行期断言：派生 deadline ≤ root deadline，violation 时 panic-in-test/错误返回。
- **验收标准：** "不从上一阶段短 deadline 派生"的回归测试：模拟误用场景被防护拦下（docs/04 §1 专门警告）。
- **依赖：** T3.2.1.1.1。
- **预估：** 1d

#### T3.2.1.1.3 cancel 全路径保证
- **目标：** 所有分支 cancel 必达；阶段结束立即取消再派生下一阶段。
- **产出：** `internal/deadline/stage_test.go`（goleak）+ 编排器使用示例测试。
- **步骤：** 1) goleak 挂入；2) 错误/提前返回/panic-recover 路径全覆盖；3) 阶段串行演示测试（对应 docs/04 §1 代码样例结构）。
- **验收标准：** 泄漏用例（故意不 cancel）被抓；正常用例零泄漏。
- **依赖：** T3.2.1.1.2。
- **预估：** 0.5d

## Story DoD

- [ ] 引擎单测覆盖 ≥ 95%；docs/04 §1 context 规则逐条对应测试名。

## 设计依据

docs/04 §1（规则+样例代码+警告）；docs/03 §3（Jev transport 约束预告）。
