# Feature 2.1.2 — 预算等式与参数校验

> 状态：done ｜ Epic 2.1 ｜ 粗估：2 人日

## 目标

实现 docs/04 §1 启动校验中与数值/预算相关的全部拒绝规则，特别是 F-001 精确等式。

## 设计要点

- 等式：`total = preliminary + first_round + rejudge + supplement + reserve` 精确相等（非四舍五入）。
- 溢出：五段相加 int64 溢出检测。
- reserve 下限（O-5 默认 200ms 可配）、宿主上限（O-4 可选 `host_max_total`）。
- provider timeout ≤ 所属阶段预算；重试/backoff 总耗时装入阶段预算。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 2.1.2.1 | 预算等式校验 | 3 |
| 2.1.2.2 | provider/策略参数校验 | 4 |

## 验收标准

- docs/04 §1 拒绝清单逐项有表驱动用例；参考配置 4800ms=400+1800+900+1100+600 通过。

## 需求关联

F-001（P0）。
