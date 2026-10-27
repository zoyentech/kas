# Feature 8.5.2 — Jev 熔断器

> 状态：done ｜ Epic 8.5 ｜ 关联：F-019 ｜ 粗估：1 人日

Jev 级熔断（连续失败 N 次 open → 快速失败直落降级链；冷却后半开单探测）。
open 期间不再为 Jev 支付 rejudge 超时预算，故障期请求时延显著下降。
