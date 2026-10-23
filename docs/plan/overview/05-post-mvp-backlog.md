# Post-MVP Backlog（v1.2 之后，不阻塞 MVP）

## 1. P1 需求（已在 docs/01 §5 定义）

| ID | 内容 | 前置 | 备注 |
|---|---|---|---|
| F-013 | 共享 cache adapter（Redis 等） | MVP 发布后 | 必须保持相同 scope、content generation、config version、TTL 与权限复核规则（docs/01 §6）；作为 `EvidenceCache` 接口的第二实现，进程内实现回归测试直接复用 |
| F-014 | 薄 HTTP/gRPC 服务包装 SDK | F-013 评估后 | 复用领域 API，不改问答语义；新增统一认证、限流、配置下发 |

## 2. 宿主接入与切换（docs/06 §6，宿主侧工程）

按 docs/06 顺序推进，均不属于 KAS 仓库 MVP：

1. **A. Shadow run**：脱敏 GoldenSet/获批抽样影子对比，不展示用户、不回灌。
2. **B. 内部 QA**：Health/metrics/fallback/cache scope/有效期/Reload 验证；禁止绕过门禁。
3. **C. 小流量灰度**：达标且 P0 安全错误为 0 后切流；监控 p95/p99、deadline、fallback、cache 隔离；P0 问题立即回切。
4. **D. 扩展调用方**：逐调用方迁移；全部通过后再评估退役旧链与 RAGFlow 职责边界（导入/管理/检索/事实源分别评估）。

宿主若为 ai-support-platform，接入前必须先复核生产 answered path（docs/06 §1：PostgreSQL script_* 事实源、RAGFlow 异步管理镜像的既有结论不得凭管理集成推翻）。

## 3. 远期候选（未承诺）

- 配置控制面对接（宿主集中下发调用 Reload 的标准 adapter 示例）。
- 回答生成的 LLM 抽象层（v1.2 仅约束回答组装器接口，生成仍归宿主）。
- 多语言绑定（非 Go 宿主经服务化包装接入，F-014 之后）。
- 决策缓存命中率与 Evidence 缓存命中率的容量规划工具（基于 MVP 指标数据）。
