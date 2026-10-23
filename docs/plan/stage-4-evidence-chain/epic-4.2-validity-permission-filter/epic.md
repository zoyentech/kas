# Epic 4.2 — 有效期与权限过滤

> 状态：todo ｜ Stage 4 ｜ 关联需求：F-003（P0）｜ 粗估：4.5 人日

## 目标与价值

实现 docs/01 §4 第 5~8 步的证据安全闸门：丢弃未授权/未生效/已过期/有效期未知的 Evidence；缓存命中后仍复核；补检结果重新过滤。有效期优先来源显式时间，缺失按批准 max-age 推导，策略缺失 fail-closed（docs/03 §4）。

## 验收（Epic DoD）

- F-003 验收行：expired/future/unknown 按策略过滤，不进答案或缓存（docs/01 §5）。
- 过滤原因计数（expired/future/unauthorized/unknown_validity）通过事件总线可观测。
- "已批准来源 max-age → 截止时间推导正确、cache TTL 不晚于截止"（docs/05 §2）有专项用例。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 4.2.1 | 有效期判定 | 4.2.1.1 过滤规则；4.2.1.2 过滤可观测 |
| 4.2.2 | 权限复核 | 4.2.2.1 scope 权限过滤 |

## 依赖

- Epic 4.1（类型）；S3（事件总线）。

## 关联设计

docs/01 §4；docs/03 §4（有效期优先级）；docs/04 §2（TTL 受有效期约束）；docs/05 §2（对应矩阵行）。
