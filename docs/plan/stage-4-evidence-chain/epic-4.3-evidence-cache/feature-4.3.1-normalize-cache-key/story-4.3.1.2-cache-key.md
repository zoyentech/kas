# Story 4.3.1.2 — cache key 组成

> 状态：todo ǀ 作为安全负责人，我希望 key 隔离维度一个不少且不留明文，以便杜绝跨权限/跨代次复用。

## 前置依赖

- T4.3.1.1.1；S2（config_version）。

## 任务

#### T4.3.1.2.1 七维 key 组合器
- **目标：** `CacheKey` 结构 + 组合函数（含 retrieval_policy_version，取自配置策略指纹）。
- **产出：** `internal/cache/key.go`。
- **步骤：** 1) 维度结构体（docs/04 §2 公式逐项）；2) retrieval_policy_version = 有效性/检索策略子集的规范化哈希；3) 组合为待哈希字节串。
- **验收标准：** 任一维度变化 → 字节串变化（表驱动七组）。
- **依赖：** 无。
- **预估：** 1d

#### T4.3.1.2.2 不可逆摘要存储
- **目标：** 存储侧只保留 key 的 SHA-256 摘要；value 不含问题原文。
- **产出：** key 摘要化 + 存储结构约定。
- **步骤：** 1) hash(key)→存储主键；2) 断言存储层无法反推原文（测试检查仅摘要存在）；3) godoc 声明隐私边界。
- **验收标准：** 存储序列化输出不含 question/身份明文（扫描测试）。
- **依赖：** T4.3.1.2.1。
- **预估：** 0.5d

#### T4.3.1.2.3 key 隔离矩阵测试与 DeleteNamespace
- **目标：** docs/05 §2 隔离行自动化：scope/epoch/namespace/generation/config_version 变化全 miss；namespace 主动清理生效。
- **产出：** `internal/cache/key_isolation_test.go`；`DeleteNamespace` API。
- **步骤：** 1) Put 后改单维 Get → miss；2) DeleteNamespace 后全 miss；3) 计数事件（S6 预埋）。
- **验收标准：** 隔离矩阵全绿；F-004 验收证据留档。
- **依赖：** T4.3.1.2.2。
- **预估：** 1d

## Story DoD

- [ ] key 面冻结；隔离矩阵与摘要化进入 CI。

## 设计依据

docs/04 §2（key 公式与"存储仅使用不可逆 key 摘要"）；docs/05 §2。
