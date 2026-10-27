# ADR-0003 — v1.3 运营洞察与诊断面边界

> 状态：proposed ｜ 日期：2026-10-27 ｜ 关联：F-015~F-019（Stage 8）；F-014（post-MVP）；ADR-0001

## 背景

v1.3 需要在 SDK 内交付：进程内运行统计（成功率/时延分位）、余额监测、本机诊断
查询 API 与单页仪表盘。docs/design/01 §1 明确 v1.2 不做"独立服务进程与
HTTP/gRPC 包装"（F-014 服务化在 post-MVP）；新增 HTTP 面必须先厘清边界。
同时 v1.2.0 审核发现统计面仅 Prometheus（宿主需自建抓取链路）、余额完全不可见、
provider panic 可击穿宿主进程。

## 决策

1. **统计在进程内，与 Prometheus 面解耦**：`internal/stats` 订阅既有事件总线，
   聚合滚动窗口与分位；F-006 的 Prometheus collector 保持不变。理由：Prometheus
   面服务规模化监控（宿主负责抓取/存储/告警）；本机排障需要零依赖直查
   （`Client.Stats()` + 诊断 API），两者数据同源（同一事件流）而非互相替代。
2. **诊断 API 是可选工具组件，不是服务化**：`WithDiagnosticServer` Option，
   默认**不启用**（不启则零 goroutine/零端口/零开销）；启用后默认绑定
   `127.0.0.1`，无鉴权、无多租户——仅本机单人诊断用途（需求方明示无需考虑
   隐私安全）。对外提供服务仍归宿主（F-014 边界不变）。
3. **余额监测 fake 先行**：`BalanceSource` 契约 + Jev adapter 按 O-2 同策略
   （协议文档未到位，以最小契约 + fake server 先行）；文档到位仅改 adapter。
4. **仪表盘单文件内嵌**：单 HTML（vanilla JS + 内联 SVG）经 `embed` 打进 SDK，
   无构建链、无 CDN、无外网依赖；体量上限 60KB。
5. **panic 隔离与熔断属稳定性补全**：`deadline.Call` recover provider panic
   （docs/design/04 §1 "provider 必须响应 ctx" 契约的补全，而非新边界）；
   Jev 熔断只作用于降级链首选级，配置可关。

## 备选方案

| 备选 | 结论 | 理由 |
|---|---|---|
| 只用 Prometheus（Local Prometheus + 查询面） | 否决 | 宿主需部署抓取/存储链路，"本机秒开"目标不可达；依赖重 |
| OTel metrics 导出 | 否决 | 引入 OTel SDK 依赖与宿主 collector 要求，超 v1.3 必要性 |
| 独立仪表盘进程/二进制 | 否决 | 违背 SDK 形态；宿主部署面变大 |
| 诊断 API 默认启用（0.0.0.0） | 否决 | 默认面最小化是 SDK 底线；本机用途无需对外 |
| SPA 用 React/Vue + 构建链 | 否决 | 构建链与分发复杂度不成比例；手写 SVG 折线/条形足够 |

## 后果

- 正面：本机零依赖排障闭环（API + 仪表盘）；故障期时延与可用性改善（熔断）；
  三方 provider 缺陷不再影响宿主存活（panic 隔离）；余额耗尽可预期。
- 负面/代价：SDK 二进制附带 ~60KB 内嵌静态资源；diag server 引入 net/http 监听
  （默认关闭规避）；stats 有固定内存占用（窗口直方图，有界恒定）。
- 缓解：默认关闭 + 仅本机绑定；体量上限写进 Story 验收；窗口内存有界有测试。

## 影响需求

| 需求 | 影响 |
|---|---|
| F-015~F-019 | 本 ADR 为其边界与口径依据 |
| F-014 | 边界不变：诊断面≠服务化，仍 post-MVP |
| F-006 | 不受影响；两数据面同源解耦 |
