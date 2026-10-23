# Story 7.1.2.1 — 安全场景套件

> 状态：done ǀ 作为项目负责人，我希望 P0 安全断言是自动化硬门禁，以便零人为疏漏。

## 前置依赖

- S4/S5 完成；Story 7.1.1.1（P0 子集）。

## 任务

#### T7.1.2.1.1 过期/跨scope/未授权零容忍套件
- **目标：** 三断言 = 0：过期 Evidence 用于答案、跨权限/租户/namespace cache hit、无权限或不存在 citation。
- **产出：** `acceptance/matrix/safety_test.go`。
- **步骤：** 1) 构造过期语料+多 scope+越权引用场景；2) 断言三类计数恒零；3) corpus generation 切换后旧代 cache 不读取。
- **验收标准：** docs/05 §2 对应四行（过期/同问不同 scope/corpus 改变/引用完整性）全绿。
- **依赖：** 无。
- **预估：** 1d

#### T7.1.2.1.2 fail-closed 与 max-age 场景
- **目标：** 有效期未知无策略 → fail-closed；批准 max-age → 截止推导正确且 TTL 不晚于截止。
- **产出：** 同套件补充场景。
- **步骤：** 1) 未知有效期语料（无策略）断言 clarify/abstain；2) max-age 推导与缓存 TTL 联动断言；3) 时间边界（推导截止=now）。
- **验收标准：** docs/05 §2 两行（unknown/max-age）自动化。
- **依赖：** T7.1.2.1.1。
- **预估：** 0.5d

## Story DoD

- [ ] P0 安全集全零断言进入 CI（作为 M6 硬门禁）。

## 设计依据

docs/05 §2 安全行、§3 P0 门槛。
