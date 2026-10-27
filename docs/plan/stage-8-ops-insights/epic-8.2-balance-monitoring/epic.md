# Epic 8.2 — 余额监测

> 状态：done ｜ Stage 8 ｜ 关联需求：F-016（P0）｜ 粗估：2 人日

## 目标与价值

Jev 是唯一计费依赖，余额耗尽表现为整链降级且难以定位。建立 BalanceSource 契约与
Jev 适配（O-2 协议未定，fake 先行），周期轮询 + 四态判定 + 阈值事件，进 Health
detail / Stats / 仪表盘。

## 验收（Epic DoD）

- 余额四态（normal/low/critical/unknown）与阈值可配置；越阈值发事件。
- 后台轮询生命周期干净（Close 停止，goleak 验证）。
- fake Jev 余额端点全链路可演示（adapter→poller→stats→API→仪表盘）。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.2.1 | BalanceSource 契约 | 8.2.1.1 balance-interface |
| 8.2.2 | Jev 适配与轮询 | 8.2.2.1 jev-balance-adapter；8.2.2.2 balance-poller |

## 依赖

- S6 事件总线与健康语义；O-2（协议文档；未到位前 fake 先行）。

## 关联设计

ADR-0003（fake 先行策略沿用 5.2）；docs/design/04 §4（Health detail）。
