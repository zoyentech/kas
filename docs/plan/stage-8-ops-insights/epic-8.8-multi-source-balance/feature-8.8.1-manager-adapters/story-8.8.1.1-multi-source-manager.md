
# Story 8.8.1.1 — 多源 Manager、适配器与热更新

> 状态：done ｜ v1.4.0 交付（commit 见 CHANGELOG）

## 交付清单

- [x] balance.Manager：命名源注册表（独立轮询/独立阈值/热 Upsert/Remove/SetConfig/Close）
- [x] adapters.DeepSeek（/user/balance 契约，多币种优先 CNY，is_available=false→unknown）
- [x] adapters.AliyunBSS（POP RPC V1 HMAC-SHA1 签名自实现，零 SDK 依赖；Qwen 计费口径）
- [x] Client：UpsertBalanceSource/RemoveBalanceSource/SetBalanceConfig/BalanceOf/BalanceSources
- [x] config：balance.sources 按源覆盖（校验合成后 critical < low）
- [x] Health：balance:<source> 按源分列
- [x] 看板：余额状态卡按 balance_source 变量 repeat + 组件余额总览表；examples 三源演示
- [x] 测试：Manager 热更新矩阵 / 适配器 fake 契约 / 验收多源 e2e（env 自动注册 + 源级阈值 + 运行时 Upsert/Remove）

## 设计依据

ai-support-platform #1243（参考实现）；ADR-0003 v1.4 修订；docs/monitoring.md §三。
