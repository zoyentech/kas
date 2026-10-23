# Story 4.2.1.1 — 有效期过滤规则

> 状态：todo ǀ 作为产品负责人，我希望过期/未生效知识绝不进入答案，以便回答永远基于当前有效事实。

## 前置依赖

- Story 4.1.1.1（类型）；S2（validity 配置）。

## 任务

#### T4.2.1.1.1 expired/future 过滤
- **目标：** 按 ValidUntil/ValidFrom 与当前时刻过滤。
- **产出：** `internal/evidence/validity/filter.go`。
- **步骤：** 1) 纯函数 `Filter(items, now, policy) (kept, reasons)`；2) 边界：ValidUntil==now 视为过期；3) exclude_future_dated 开关。
- **验收标准：** 边界时刻表驱动用例（±1ns）；future 仅在开关开启时过滤（默认 true）。
- **依赖：** 无。
- **预估：** 0.5d

#### T4.2.1.1.2 unknown fail-closed
- **目标：** 无显式时间且无批准策略的证据按未知处理并丢弃。
- **产出：** 同文件 unknown 分支。
- **步骤：** 1) require_validity_policy_for_each_source=true 时未知即滤；2) false 时仍要求显式批准才放行（实现为：未知一律滤除，除非策略显式 allow_unknown，配置默认不启用并记 ADR 说明）；3) 原因码 unknown_validity。
- **验收标准：** docs/05 §2"有效期未知且无来源策略 → fail-closed"用例通过。
- **依赖：** T4.2.1.1.1。
- **预估：** 1d

#### T4.2.1.1.3 max-age 推导截止时间
- **目标：** 来源有批准 max-age 时：截止 = ObservedAt + max_age。
- **产出：** 推导函数 + 与 TTL 联动的输出（derived ValidUntil）。
- **步骤：** 1) 查策略表（default_max_age per source）；2) 推导后仍走 expired 判定；3) 输出推导值供缓存 TTL 使用。
- **验收标准：** docs/05 §2"已批准来源 max-age"行：推导正确且 cache TTL 不晚于截止（与 4.3.2 联测）。
- **依赖：** T4.2.1.1.2。
- **预估：** 1d

## Story DoD

- [ ] 三类规则表驱动全覆盖；推导截止时间输出到缓存路径。

## 设计依据

docs/03 §4（有效期优先级与 fail-closed）；docs/02 validity 节；docs/05 §2。
