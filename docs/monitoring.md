# KAS 监控接入指南（v1.3，路线 A：Prometheus + Grafana）

> 决策依据 [ADR-0003](adr/0003-ops-insights-boundary.md)：展示层由 Prometheus+Grafana
> 栈承载；KAS 侧仅新增余额采集与熔断（gauge 已导出）。历史数据由 Prometheus
> 保留（默认 30 天，可调）。

## 一、宿主接入（约 10 行代码）

KAS 不开 listener——宿主用 `WithPrometheusRegisterer` 注入 registry，再以
`promhttp` 暴露 `/metrics`：

~~~go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    kas "github.com/zoyentech/kas"
)

client, err := kas.New(cfg, kas.Dependencies{
    EvidenceProvider: adapter,
}, kas.WithPrometheusRegisterer(prometheus.DefaultRegisterer))

mux := http.NewServeMux()
mux.Handle("/metrics", promhttp.Handler())
http.ListenAndServe("0.0.0.0:8080", mux) // 监控面端口归宿主管理
~~~

完整可运行演示（含 fake Jev、余额递减、持续流量）：`go run ./examples/metrics`。

## 二、一键起监控栈

~~~bash
# 在仓库 monitoring/ 目录内执行（zsh 交互模式不剥离行尾 # 注释，命令勿带行尾注释）
cd ~/zoyen/KAS/kas/monitoring
docker compose up -d
~~~

- Grafana http://127.0.0.1:23000 （匿名只读；默认首页即 **KAS Overview** 看板）
- Prometheus http://127.0.0.1:29090 （保留 30 天，`docker-compose.yml` 可调）
- 抓取目标默认 `host.docker.internal:8080`（改 `prometheus/prometheus.yml`
  的 targets 指向宿主应用；抓取间隔默认 5s）
- 演示流量（另开终端，**在仓库根目录**执行）：`cd ~/zoyen/KAS/kas && go run ./examples/metrics`
- 停止栈：`docker compose down`（数据卷保留）

> 端口提示：栈只绑定 IPv4 `127.0.0.1`。若本机 `::1:3000` 已有隧道/转发，
> 请用显式 `http://127.0.0.1:23000` 访问，避免 `localhost` 被 IPv6 解析到其他服务；
> 端口冲突时可改 `docker-compose.yml` 的 ports 映射。

看板面板：组件健康卡（Evidence/Jev）、余额状态卡、熔断状态卡、成功率、QPS、
请求时延 p50/p95/p99、阶段耗时 p95、阶段超时、缓存命中率、降级路由、
Provider 错误、余额趋势、证据过滤。

## 三、余额监测（F-016）

Prometheus 是拉模式——余额采集内建于 SDK（后台轮询 + gauge 导出）：

~~~yaml
decision:
  provider: jev
  jev:
    endpoint_env: KAS_JEV_ENDPOINT
    api_key_env: KAS_JEV_API_KEY
    balance_endpoint_env: KAS_JEV_BALANCE_ENDPOINT   # GET，Bearer 同 key
balance:
  enabled: true
  interval: 60s
  low_threshold: 300       # amount ≤ low 且 > critical → low
  critical_threshold: 50   # amount ≤ critical → critical
~~~

- 四态：normal / low / critical / unknown（查询失败；保留上次成功值并退避重试）。
- 指标：`kas_balance_amount{source,unit}`、`kas_balance_state{source}`
  （3=normal 2=low 1=critical 0=unknown）；迁移同时发内部 BalanceEvent。
- `Client.Balance()` 可编程查询；Health 报告含 balance 信息组件。
- 非 Jev 计费源：注入 `Dependencies.BalanceSource`（实现
  `Balance(ctx) (BalanceInfo, error)`）即可复用同一监测面。
- 端点契约（O-2 未定，最小契约先行）：`GET {endpoint}` →
  `{"amount": 42.5, "unit": "credit"}`；协议文档到位仅改 `internal/balance/jev.go`。

## 四、Jev 熔断（F-019，默认开启）

~~~yaml
decision:
  jev:
    breaker:
      enabled: true          # 默认开；false 完全关闭
      failure_threshold: 5   # 连续失败 5 次进 open
      cooldown: 30s          # 冷却期满转 half_open 单探测
~~~

- open 期间请求**快速失败**直接走降级链（Jev→规则→决策缓存→abstain），
  不再为 Jev 支付 rejudge 超时预算；探测成功自动回 closed。
- 指标：`kas_provider_breaker_state{provider}`（0=closed 1=open 2=half_open）；
  Health detail 同步展示（`jev` 组件 `breaker=open` 等）。

## 五、panic 隔离（F-019）

三方 provider（Evidence/Decision/Preliminary/Assembler）panic 不再击穿宿主
进程：`deadline.Call`、降级链每级、阶段挂载点、组装器四层守卫把 panic 归一为
契约违规（`ErrProviderContract`），请求走降级路径，事件计数
`kas_provider_errors_total{error_class="panic"}`。panic 值原文不进入任何输出面
（可能含敏感数据）；provider 作者应自行包装自身 panic 做调试。

## 六、PromQL 速查

| 目标 | 查询 |
|---|---|
| 成功率 % | `100 * sum(rate(kas_requests_total{outcome=~"ok\|degraded"}[5m])) / clamp_min(sum(rate(kas_requests_total[5m])), 1e-9)` |
| QPS | `sum(rate(kas_requests_total[5m]))` |
| 请求 p95 | `histogram_quantile(0.95, sum by (le) (rate(kas_request_duration_seconds_bucket[5m])))` |
| 阶段 p95 | `histogram_quantile(0.95, sum by (le, stage) (rate(kas_stage_duration_seconds_bucket[5m])))` |
| 阶段超时/s | `sum by (stage) (rate(kas_stage_deadline_exceeded_total[5m]))` |
| 缓存命中率 % | `100 * sum(rate(kas_evidence_cache_total{result="hit"}[5m])) / clamp_min(sum(rate(kas_evidence_cache_total{result=~"hit\|miss"}[5m])), 1e-9)` |
| 降级路由 | `sum by (route, reason) (increase(kas_decision_fallback_total[1h]))` |
| Provider 错误 | `sum by (provider, error_class) (rate(kas_provider_errors_total[15m]))` |
| 证据过滤 | `sum by (reason) (rate(kas_evidence_filtered_total[5m]))` |
| 余额 | `kas_balance_amount{source="jev"}` |
| 余额状态 | `kas_balance_state{source="jev"}`（3/2/1/0） |
| 熔断状态 | `kas_provider_breaker_state{provider="jev"}`（0/1/2） |

口径说明：abstain/ask_clarification 属正确行为计入成功（outcome=ok）；degraded
单列（`kas_requests_total{outcome="degraded"}`）不折算失败。label 白名单见
配置 `metrics.label_allowlist`（新增 `source`/`unit` 供余额 gauge）。

## 七、告警（归宿主）

本栈不含 Alertmanager。建议告警（在宿主现有告警体系配置上述 PromQL）：

- 成功率 5 分钟 < 95%
- `kas_provider_breaker_state == 1` 持续 5 分钟
- `kas_balance_state <= 1`（low/critical）
- `kas_provider_errors_total{error_class="panic"}` increase > 0
- `kas_stage_deadline_exceeded_total` rate 激增
