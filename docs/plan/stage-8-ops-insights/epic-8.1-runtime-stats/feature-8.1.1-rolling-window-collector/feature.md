# Feature 8.1.1 — 滚动窗口聚合器

> 状态：todo ｜ Epic 8.1 ｜ 关联：F-015 ｜ 粗估：1.5 人日

订阅事件总线，把 RequestEnd/StageEnd/Cache/Filter/Fallback/ProviderError 聚合为
多窗口（1m/5m/15m/1h）环形桶计数与对数直方图。并发安全、内存有界。
