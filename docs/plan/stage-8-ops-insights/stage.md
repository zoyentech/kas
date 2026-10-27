
# Stage 8 — 运营洞察与接入稳定化（v1.3）

> 状态：done ｜ 前置：S1~S7（已完成，v1.2.0 已发布）｜ 里程碑：M7 ｜ 粗估：8.5 人日

## 路线决策（2026-10-27，项目负责人确认）

**展示层走 Prometheus + Grafana（路线 A）**：接入 KAS SDK 的服务愿意常驻
Prometheus+Grafana；宿主应用方便暴露 `/metrics`；历史数据有价值（天/周级保留）。
进程内统计（8.1）、诊断 API（8.3）、内嵌 SPA（8.4）**延迟为可选 Epic**——
触发条件：未来需要"零依赖秒开"的本机诊断页时再实施（届时诊断 server 可顺带
暴露 /metrics 兼作抓取目标）。

## 背景（v1.2.0 进展审核结论）

v1.2.0 已发布：P0 F-001~F-012 全落地、CI 7 job 全绿、GoldenSet 验收 PASS。
对照运营目标审核结论：

1. **数据面已就绪**：11 个指标族已导出（成功率/时延直方图/阶段/缓存/过滤/
   降级/健康均可 PromQL 查询）；缺的是"开箱即看的栈与看板"，不是指标本身。
2. **无余额/配额监测**：Jev 是唯一计费依赖；余额耗尽表现为整链降级难定位。
   Prometheus 是拉模式——余额采集（轮询器）必须内建于 SDK，展示走 gauge。
3. **稳定性薄弱点**：
   - `deadline.Call` 无 recover：任意 provider panic 击穿宿主进程（全库唯一
     recover 在事件总线订阅者）；
   - Jev 无熔断：故障期每请求烧满 rejudge 预算才降级（慢失败）；
   - 状态盲区：config_version 类低频状态不适合指标 label（基数纪律，
     docs/design/04 §5），由 Health/trace 承载（已就绪），熔断态/余额态
     用 gauge 值承载（本阶段新增）。

## 目标

1. 接入稳定化：provider panic 隔离（不击穿宿主）、Jev 熔断（open 期快速降级）（F-019）。
2. 余额监测：BalanceSource 契约 + Jev 适配（O-2 fake 先行）+ 四态 + 阈值事件 + gauge 导出（F-016）。
3. 监控栈交付：docker compose 一键起 Prometheus+Grafana、预置 kas-overview
   看板（成功率/QPS/时延分位/阶段/缓存/降级/错误/余额/熔断）、宿主 /metrics
   示例、监控指南（F-015/F-017/F-018 由该栈承载）。
4. 验收与 v1.3.0 发布。

## 范围内 / 范围外

**范围内**：panic 守卫（deadline.Call/降级链阶段/runStage/组装器）；Jev 熔断
（默认开、可配可关）；internal/balance（契约/Jev adapter/轮询器）；余额与熔断
gauge；monitoring/ 栈（compose+prometheus.yml+Grafana provisioning+看板 JSON）；
examples/metrics；docs/monitoring.md；验收与发布。

**范围外（deferred）**：进程内滚动窗口统计（8.1）、诊断 HTTP API（8.3）、
内嵌单页仪表盘（8.4）——编号与文档保留，实施待触发；对外服务化（F-014 不变）；
告警外发（Alertmanager 归宿主栈，看板与查询手册先行）；余额操作类接口。

## 出口条件（DoD）

- [x] 注入 panic 的 evidence/decision/preliminary/assembler 全部：请求降级返回、
      事件 error_class=panic、进程存活（-race）。
- [x] 熔断三态迁移矩阵测试；open 期间 fake server 调用数为 0；恢复半开自动回闭；
      open 期请求时延远小于 rejudge 预算；gauge 可见。
- [x] 余额四态 + 阈值边界测试；unknown 保留上次成功值；fake Jev 全链路
      （adapter→poller→事件→gauge）演示；Health 含余额信息组件。
- [x] monitoring/ 栈 `docker compose up` 即用；看板 JSON 结构校验测试进 CI；
      examples/metrics 可运行并被抓取。
- [x] 全量 -race、golangci v2、gofumpt、GoldenSet 不回归；v1.3.0 tag 走发版流水线全绿。

## Epic 列表

| Epic | 名称 | Feature | 粗估 | 状态 |
|---|---|---|---|---|
| 8.5 | 接入稳定化 | 8.5.1 panic 隔离；8.5.2 Jev 熔断；8.5.3 稳定性配套 | 3 人日 | active |
| 8.2 | 余额监测 | 8.2.1 契约；8.2.2 Jev 适配与轮询（gauge 导出） | 2 人日 | active（数据面改 gauge） |
| 8.7 | 监控栈交付 | 8.7.1 compose+看板+示例+指南 | 2 人日 | active（新增，替代 8.1/8.3/8.4 展示层） |
| 8.6 | 验收与发布 | 8.6.1 稳定性/余额/看板验收；8.6.2 v1.3.0 | 1.5 人日 | active（内容按路线 A 调整） |
| 8.1 | 运行统计内核 | （见原文档） | — | **deferred** |
| 8.3 | 诊断查询 API | （见原文档） | — | **deferred** |
| 8.4 | 单页仪表盘 | （见原文档） | — | **deferred** |

## 需求映射（按路线 A 修订）

| ID | 优先级 | 摘要 | 落点 |
|---|---|---|---|
| F-015 | P0 | 运行观测：Prometheus 栈 + 预置看板 + PromQL 手册承载 | 8.7 |
| F-016 | P0 | 余额监测：BalanceSource、Jev 适配（fake 先行）、四态 gauge、阈值事件 | 8.2 |
| F-017 | P0 | 查询面：Prometheus/Grafana（PromQL/看板即查询 API） | 8.7 |
| F-018 | P0 | 展示面：Grafana 预置 kas-overview 看板 | 8.7 |
| F-019 | P0 | 接入稳定化：panic 隔离、Jev 熔断 | 8.5 |

## 关键口径（随路线决策更新）

- 成功率 PromQL：`sum(rate(kas_requests_total{outcome=~"ok|degraded"}[5m])) /
  clamp_min(sum(rate(kas_requests_total[5m])), 1e-9)`（abstain/clarify 属正确行为，
  degraded 单列不折算——outcome 枚举已按此设计）。
- 时延分位：`histogram_quantile`（导出直方图桶 5ms~10s，请求总预算在界内）。
- 余额 gauge：`kas_balance_amount{source,unit}`（值）+ `kas_balance_state{source}`
  （值 3=normal/2=low/1=critical/0=unknown）；状态变更发 BalanceEvent。
- 熔断 gauge：`kas_provider_breaker_state{provider}`（0=closed/1=open/2=half_open），
  迁移发 BreakerEvent；Health detail 同步展示。
- 熔断参数：默认开启；连续失败 5 进 open，冷却 30s 后半开单探测；配置可关。
- 余额口径：amount > low → normal；> critical → low；≤ critical → critical；
  查询失败 → unknown（保留上次成功值并标记失联时长）。
- 新增 label：`source`、`unit` 进 label allowlist（有界：余额来源与单位枚举）。

## 风险与注意事项

- O-2 余额端点协议未定 → fake 先行（与 5.2 同策略），协议到位仅改 internal/balance 的 Jev adapter。
- Grafana/Prometheus 版本固定在 monitoring/docker-compose.yml（避免 latest 漂移）。
- 看板 JSON 手写，无 CI 渲染验证 → 结构校验测试（面板/表达式/数据源齐全）兜底。
- 余额金额单位因计费源而异 → unit label 有界枚举，不允许自由字符串增长。

## 实现顺序

8.5.1（panic，真实风险最优先）→ 8.5.2（熔断）→ 8.2（余额）→ 8.7（监控栈）→ 8.6（验收+发布）。
