# Epic 8.4 — 单页仪表盘

> 状态：todo ｜ Stage 8 ｜ 关联需求：F-018（P0）｜ 粗估：2 人日

## 目标与价值

内嵌单文件 SPA（vanilla JS + 内联 SVG，无构建无 CDN 无外网），本机打开 `/` 即看：
总状态卡、成功率/QPS 趋势、时延分位、阶段耗时、缓存与过滤、降级路由、余额趋势。
5s 轮询自动刷新。

## 验收（Epic DoD）

- 打开即渲染（file:// 不需要；经 diag server `/`）；断网刷新仍渲染（资源全内嵌）。
- 数据全来自 /api/overview（一次拉全）；Close 后数据停更且不崩溃。
- 单 HTML 文件 < 60KB；`go vet`/lint 对 embed 无告警。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.4.1 | 内嵌 SPA | 8.4.1.1 dashboard-page；8.4.1.2 dashboard-panels |

## 依赖

- Epic 8.3（/ 与 /api）；8.1/8.2 数据。

## 关联设计

ADR-0003（单文件内嵌、无第三方库决策）。
