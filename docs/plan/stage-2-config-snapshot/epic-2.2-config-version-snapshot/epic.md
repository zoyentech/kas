# Epic 2.2 — config_version 与原子 Reload

> 状态：todo ｜ Stage 2 ｜ 关联需求：F-011（P0）、F-009 部分 ｜ 粗估：4 人日

## 目标与价值

实现"同一快照内一切版本一致"的机制：脱敏 config_version 生成；不可变 Runtime Snapshot 构建与 `atomic.Pointer` 原子替换；Reload 任何一步失败保持旧值；在途请求固定旧快照（docs/01 §6 发布顺序）。

## 验收（Epic DoD）

- F-011：并发场景（Answer 进行中 Reload）下，单请求全程只见一个 config_version；失败 Reload 后版本不变。
- secret 明文绝不参与版本计算与错误输出（docs/01 §6）。
- 相同配置重复加载产生相同 version；字段顺序无关。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 2.2.1 | 脱敏 config_version | 2.2.1.1 规范化与哈希 |
| 2.2.2 | 原子 Reload 与 snapshot | 2.2.2.1 Runtime Snapshot；2.2.2.2 原子替换语义 |

## 依赖

- Epic 2.1（Load/Validate）。

## 关联设计

docs/01 §6（发布顺序、版本口径）；docs/03 §5（Client.Reload/New 语义）。
