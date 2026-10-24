# Story 2.1.2.2 — provider/策略参数校验

> 状态：done ｜ 作为配置维护者，我希望 provider 与策略参数的错误在加载期暴露，而非运行期炸出。

## 前置依赖

- Story 2.1.2.1（规则框架复用）。

## 任务

#### T2.1.2.2.1 provider timeout ≤ 阶段预算
- **目标：** 校验 decision.jev 等 provider 超时不晚于所属阶段预算。
- **产出：** provider timeout 规则。
- **步骤：** 1) 定义 provider→阶段映射（jev→rejudge/初判相关阶段，依 S3 编排语义固化映射表）；2) 超出即拒绝；3) 映射表写注释并留 ADR 引用。
- **验收标准：** provider timeout 超阶段预算被拒；等于边界值通过。
- **依赖：** T2.1.2.1.1。
- **预估：** 1d

#### T2.1.2.2.2 重试/backoff 装入预算
- **目标：** 若配置重试，其总最坏耗时必须装入对应阶段预算。
- **产出：** retry/backoff 规则。
- **步骤：** 1) 最坏耗时 = Σ(attempt_i + backoff_i)；2) 装不下即拒绝；3) max_attempts=1 时规则自动满足。
- **验收标准：** max_attempts=3 且超预算被拒；=1 通过（docs/03 §3 默认单次）。
- **依赖：** T2.1.2.2.1。
- **预估：** 0.5d

#### T2.1.2.2.3 有效期策略完整性
- **目标：** 校验 validity 配置自洽。
- **产出：** validity 规则集。
- **步骤：** 1) `require_validity_policy_for_each_source=true` 时 default_max_age 必须非空且 key 为合法 source；2) max-age 值为正 duration；3) `fail_closed_on_unknown_validity` 与 exclude_future_dated 为布尔（类型层已保证，语义层复查组合）。
- **验收标准：** 缺策略/非法 max-age 被拒；docs/02 的两种来源样例通过。
- **依赖：** T2.1.1.1.1。
- **预估：** 0.5d

#### T2.1.2.2.4 缓存容量参数校验
- **目标：** 校验 evidence.cache 与 decision.fallback.cache 容量参数。
- **产出：** cache 参数规则集。
- **步骤：** 1) max_entries>0、max_bytes>0、max_item_bytes>0 且 ≤ max_bytes；2) ttl>0；3) decision cache 开关与 include_* 布尔组合合法。
- **验收标准：** 非法容量（0/负/item>total）被拒；docs/02 数值通过。
- **依赖：** T2.1.1.1.1。
- **预估：** 0.5d

## Story DoD

- [x] docs/04 §1"拒绝"清单数值类全部有自动化用例并计入 CI。

## 设计依据

docs/04 §1（启动校验清单）；docs/02（参数样例）；docs/03 §3（max_attempts 语义）。
