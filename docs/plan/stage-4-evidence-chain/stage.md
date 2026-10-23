# Stage 4 — Evidence 链路

> 状态：done ｜ 前置：S3 ｜ 里程碑：M4（与 S6 共同达成）｜ 粗估：12 人日

## 目标

实现 Evidence 侧全部 P0 能力：EvidenceProvider 契约与参考实现/故障注入（F-003 前置）、知识有效期与权限过滤（F-003）、问题规范化与安全缓存 key、有界内存缓存全部策略（F-004）。缓存命中仍复核当前权限与有效期；补检结果重新过滤。

## 范围内

EvidenceProvider/Evidence/EvidenceQuery 类型与契约测试套件；in-memory 参考实现与故障注入包装器；expired/future/unknown fail-closed/max-age 推导过滤；权限判定（声明式映射+宿主注入函数）；NFKC 规范化；cache key 组成与不可逆摘要；LRU 字节/条目上限与 TTL min 规则；读写路径（miss 回源/hit 复核/不足量回源）。

## 范围外

真实检索服务接入（宿主侧 adapter，post-MVP 示例可放 docs）；决策链（S5）；决策缓存（S5 Epic 5.3）。

## 入口条件

- S3 出口条件满足（挂载点/事件总线可用）。

## 出口条件（DoD）

- [ ] F-003：expired/future/unknown 三类被过滤且不进答案或缓存（docs/01 §4 第 6 步）；fail-closed 用例齐备。
- [ ] F-004：跨 scope/epoch/namespace/generation/config_version 的 cache hit 为 0（docs/05 §2 对应行）。
- [ ] 缓存 TTL ≤ min(配置 TTL, 来源新鲜度, 最早 ValidUntil-now)；≤0 不写入。
- [ ] 规范化属性测试：否定词/数字/单位/型号符号保留（docs/04 §2）。
- [ ] 补检结果重新过滤的组合测试通过。

## Epic 列表

| Epic | 名称 | Feature |
|---|---|---|
| 4.1 | EvidenceProvider 契约与参考实现 | 4.1.1 接口与契约测试 |
| 4.2 | 有效期与权限过滤 | 4.2.1 有效期判定；4.2.2 权限复核 |
| 4.3 | Evidence 安全缓存 | 4.3.1 规范化与 key；4.3.2 缓存策略 |

## 需求覆盖

F-003（4.2）、F-004（4.3）、F-012（缓存并发安全）。

## 风险与注意事项

- cache key 是 P0 安全面：任何 key 维度缺失都可能造成跨权限复用，维度清单必须与 docs/04 §2 逐一对照评审。
- 有效期策略组合（显式时间 vs max-age 推导 vs 未知）易出边界错，表驱动全覆盖。
