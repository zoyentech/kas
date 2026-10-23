# Feature 4.3.2 — 缓存策略

> 状态：done ｜ Epic 4.3 ｜ 粗估：2.5 人日

## 目标

有界内存缓存实现（docs/03 §6 EvidenceCache 接口）与读写路径策略：TTL min 规则、禁缓存清单、确定性淘汰、hit 复核、不足量回源。

## 设计要点

- 上限三件套：max_entries / max_bytes / max_item_bytes（docs/02）；超限确定性 LRU 淘汰并计数。
- TTL = min(配置 TTL, 来源新鲜度, 最早 ValidUntil − now)；≤0 不写入。
- 不缓存：空结果、provider error、timeout、权限不一致、有效期未知（docs/04 §2）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 4.3.2.1 | 有界内存缓存 | 3 |
| 4.3.2.2 | 读写路径 | 3 |

## 验收标准

- 淘汰确定性：相同操作序列结果一致；eviction reason 计数。
- hit→复核→（不足量）回源 全链行为符合 docs/04 §2。

## 需求关联

F-004（P0）、F-012（并发安全）。
