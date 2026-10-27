# KAS 监控栈（Prometheus + Grafana）

一键起栈与使用步骤；完整指南（宿主接入/余额/熔断/PromQL）见
[../docs/monitoring.md](../docs/monitoring.md)。

## 使用步骤

```bash
# 1. 起栈（在本目录执行；命令不要带行尾 # 注释——zsh 交互模式不剥离注释）
cd ~/zoyen/KAS/kas/monitoring
docker compose up -d
```

- Grafana：http://127.0.0.1:23000 （匿名只读，默认首页即 **KAS Overview** 看板；
  管理入口 admin/admin）
- Prometheus：http://127.0.0.1:29090 （数据保留 30 天）

```bash
# 2. 起宿主演示（回到仓库根目录！产生流量与余额递减，看板即时出数）
cd ~/zoyen/KAS/kas
go run ./examples/metrics
```

```bash
# 3. 停止 / 清理
docker compose stop          # 停止（保留数据）
docker compose down          # 移除容器（数据卷保留）
docker compose down -v       # 连数据卷一起清空
```

## 常见问题

- **`no such service: #`**：命令后带了行尾注释。zsh 交互模式默认不把 `#` 当注释，
  会把它作为参数传给 docker compose——去掉行尾注释再执行。
- **`directory not found`（go run）**：`examples/metrics` 在仓库根目录下，
  请先 `cd ~/zoyen/KAS/kas` 再 `go run ./examples/metrics`。
- **打开 localhost:3000 不是本看板**：本栈只绑 IPv4 `127.0.0.1`；若 `::1:3000`
  被隧道占用，`localhost` 可能走 IPv6 打到其他服务——用显式
  `http://127.0.0.1:23000`。
- **改抓取目标/端口**：编辑 `prometheus/prometheus.yml` 的 targets，或
  `docker-compose.yml` 的 ports；改完 `docker compose up -d` 生效。
