# Feature 3.2.1 — context deadline 规则

> 状态：done ｜ Epic 3.2 ｜ 粗估：3 人日

## 目标

deadline 计算与派生的全部规则集中在一个小包内，成为编排器唯一的时间授权方。

## 设计要点

- `minDeadline(parent, t)`：caller 更早时不可延长（docs/04 §1）。
- 阶段工厂：输入 root + 阶段名 + 预算 → 返回派生 ctx+cancel；签名上不暴露上一阶段 ctx（防误用）。
- `WithDeadlineCause` 映射 ErrDeadlineExceeded 语义（阶段/根区分）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 3.2.1.1 | root/阶段 deadline 计算 | 3 |
| 3.2.1.2 | provider ctx 合规防护 | 3 |

## 验收标准

- 全部路径（含提前返回/错误分支）cancel 均被调用（goleak+防御性测试）。
- 可选阶段未运行时不重分配预算，root 仍为上限（docs/04 §1）。

## 需求关联

F-002（P0）。
