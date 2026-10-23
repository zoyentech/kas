# Epic 6.1 — Health

> 状态：done ｜ Stage 6 ｜ 关联需求：F-005（P0）｜ 粗估：1.5 人日

## 目标与价值

`Client.Health(ctx)` 聚合组件健康：not_ready / degraded / healthy 三态语义（docs/04 §4）；短独立 check timeout；不执行用户查询、不访问 Evidence cache、不运行模型生成。

## 验收（Epic DoD）

- docs/05 §2"Health dependency 故障"行自动化：required→not_ready、optional Jev→degraded。
- check_timeout（docs/02 health 节）生效，组件探测并发执行。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 6.1.1 | 组件健康与聚合 | 6.1.1.1 Health 实现 |

## 依赖

- S3（组件 Health 接口、deadline 引擎）。

## 关联设计

docs/03 §5（Health 语义）；docs/04 §4；docs/02 health 节。
