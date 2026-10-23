# Feature 4.2.2 — 权限复核

> 状态：done ｜ Epic 4.2 ｜ 粗估：1.5 人日

## 目标

Evidence 的授权闸门：principal scope 与 PermissionTags 的判定（O-7 双模式：声明式映射默认 + 宿主注入判定函数）；缓存命中后的当前权限复核；补检结果重新过滤。

## 设计要点

- 判定输入：Evidence.PermissionTags × PrincipalScope（含 AuthorizationEpoch 语义：epoch 变化 → scope fingerprint/key 变化）。
- 缓存命中后复核 = 用当前 scope 重新判定已缓存 Evidence（docs/04 §2 缓存规则）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 4.2.2.1 | scope 权限过滤 | 3 |

## 验收标准

- docs/05 §2"同问不同 tenant/scope/epoch → 跨 scope cache hit = 0"（与 4.3 联测）。
- 补检结果重过滤组合测试（docs/01 §4 第 8 步）。

## 需求关联

F-003/F-004 交汇点；docs/01 §6 安全约束。
