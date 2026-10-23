# Feature 4.2.1 — 有效期判定

> 状态：todo ｜ Epic 4.2 ｜ 粗估：3 人日

## 目标

纯函数化的有效期过滤核心：输入 Evidence 集 + 时刻 + 策略，输出（有效集, 逐条过滤原因）。

## 设计要点

- 判定序：ValidUntil < now → expired；ValidFrom > now → future（受 exclude_future_dated）；两者 nil → 未知：仅当来源有批准 max-age 时用 ObservedAt+max_age 推导截止，否则 fail-closed（docs/03 §4）。
- 推导出的截止时间同时作为 cache TTL 上界（传给 4.3.2）。
- 全部规则表驱动可审计。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 4.2.1.1 | 过滤规则 | 3 |
| 4.2.1.2 | 过滤可观测 | 1 |

## 验收标准

- docs/05 §2 三行（过期/未知 fail-closed/max-age 推导）全部自动化。

## 需求关联

F-003（P0）。
