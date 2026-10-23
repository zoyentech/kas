# Epic 4.3 — Evidence 安全缓存

> 状态：done ｜ Stage 4 ｜ 关联需求：F-004（P0）、F-012 ｜ 粗估：4.5 人日

## 目标与价值

实现 docs/04 §2 的进程内 Evidence 缓存：key 由七维组成且只存不可逆摘要；TTL 受有效期约束；有界确定性淘汰；只缓存 Evidence 列表与元数据、绝不缓存最终回答；命中后复核权限与时效、不足量回源。

## 验收（Epic DoD）

- F-004：key 隔离矩阵全绿（scope/epoch/namespace/generation/config_version 任一变化 → miss）。
- 禁缓存清单（空结果/error/timeout/权限不一致/未知有效期）逐项用例。
- 淘汰确定性（同插入序 → 同淘汰序）与计数事件。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 4.3.1 | 规范化与 cache key | 4.3.1.1 问题规范化；4.3.1.2 key 组成 |
| 4.3.2 | 缓存策略 | 4.3.2.1 有界内存缓存；4.3.2.2 读写路径 |

## 依赖

- Epic 4.2（过滤与权限复核）；S2（config_version）。

## 关联设计

docs/04 §2 全节；docs/01 §3（Evidence Cache 组件）；docs/05 §2（corpus generation/跨 scope 行）。
