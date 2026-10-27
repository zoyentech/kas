
# Story 8.7.1.1 — compose 监控栈

> 状态：done ｜ 作为接入方，我希望一条命令起好监控栈，以便零手工配置看到 KAS 指标。

## 前置依赖

- 无（抓取目标为宿主 /metrics）。

## 任务

#### T8.7.1.1.1 compose 与抓取配置
- **目标：** monitoring/docker-compose.yml（prometheus+grafana 版本固定）+ prometheus.yml
  （抓取 host app，5s 间隔；linux host.docker.internal 兼容注释）+ grafana datasource/dashboard
  自动 provisioning。
- **产出：** monitoring/{docker-compose.yml,prometheus/prometheus.yml,grafana/provisioning/...}。
- **验收标准：** compose up 后 :9090/:3000 可用；datasource uid 固定（看板引用）。
- **依赖：** 无。
- **预估：** 0.5d

## Story DoD

- [x] 一键起栈可用；README 步骤化。

## 设计依据

ADR-0003。
