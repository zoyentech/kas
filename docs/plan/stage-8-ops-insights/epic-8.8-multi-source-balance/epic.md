
# Epic 8.8 — 多源余额与热更新（v1.4 增量）

> 状态：done ｜ Stage 8 增量 ｜ 关联需求：F-016 扩展 ｜ 粗估：2 人日
>
> 背景：v1.3 余额监测仅单源（jev）；看板"余额状态"未标明组件。KAS 未来依赖
> OpenViking/RAGFlow/Jev/DeepSeek/Qwen，其中多个付费，需按源分列余额。
> 参考实现：ai-support-platform #1243 管理监控页（DeepSeek /user/balance 契约、
> 阿里云 BSS 账户现金余额=Qwen 计费口径、Jev 无官方余额接口）。

## 验收（Epic DoD）

- 多源注册（内置 jev/deepseek/aliyun 按 env 自动 + 宿主注入 host 源）；源级阈值覆盖。
- 运行时热更新：UpsertBalanceSource / RemoveBalanceSource / SetBalanceConfig
  （宿主管理后台换凭据/调阈值不重启）；goleak 干净。
- 看板按 source 分列（余额状态卡 repeat + 组件余额总览表 + balance_source 变量）。
- Health 信息组件按源分列 balance:<source>。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 8.8.1 | Manager 与适配器 | 8.8.1.1 multi-source-manager（含适配器/热更新/看板/文档，随实现一并交付） |

## 关联设计

ADR-0003（v1.4 修订）；docs/monitoring.md §三。
