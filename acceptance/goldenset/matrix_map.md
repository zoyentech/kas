# docs/design/05 §2 验收矩阵（19 行）↔ 测试映射（Story 7.1.2.3 T2）

| # | 场景 | 测试 |
|---|---|---|
| 1 | 标准可答、问法改写 | matrix/safety_test.go TestRowStandardAnswerable |
| 2 | 空结果/低相关/no-answer | matrix/safety_test.go TestRowNoAnswer |
| 3 | Evidence 已过期/尚未生效 | matrix/safety_test.go TestRowExpiredAndCitationIntegrity |
| 4 | 有效期未知且无来源策略 | matrix/safety_test.go TestRowUnknownValidityFailClosed |
| 5 | 已批准来源 max-age | matrix/safety_test.go TestRowMaxAgeDerivationAndTTL + internal/cache TTL 联测 |
| 6 | 同问不同 tenant/scope/epoch | matrix/safety_test.go TestRowCrossScopeIsolation + internal/cache 隔离矩阵（F-004 全 miss） |
| 7 | corpus generation 改变 | matrix/safety_test.go TestRowCorpusGenerationSwitch |
| 8 | Reload 并发 | matrix/reload_test.go TestRowReloadConcurrent + internal/runtime reload_concurrent_test |
| 9 | Reload 配置无效 | matrix/reload_test.go TestRowReloadInvalid |
| 10 | Jev 合法响应 | matrix/jev_test.go TestRowJevValid |
| 11 | Jev timeout/5xx | matrix/jev_test.go TestRowJevFaults/timeout、/http_500 |
| 12 | Jev 非法 action/越界 ID/NaN | matrix/jev_test.go TestRowJevFaults/illegal_action、/out_of_range_target、/nan_confidence |
| 13 | 多条规则冲突 | matrix/rules_cache_test.go TestRowRulesConflictStable + rules 单测 |
| 14 | 决策缓存命中 | matrix/rules_cache_test.go TestRowDecisionCacheHit + dcache 单测（含伪造条目拒用） |
| 15 | provider 阻塞 | matrix/deadline_test.go TestRowProviderBlocked、TestRowRootExpiredNoCalls、TestRowCallerDeadlineNotExtended |
| 16 | Health dependency 故障 | matrix/mixed_test.go TestRowHealthFaults + obs/health 判定矩阵 |
| 17 | Metrics 成功/超时/cache/fallback | matrix/mixed_test.go TestRowMixedConcurrencyAndMetrics + obs/metrics 增量断言 |
| 18 | 引用完整性 | matrix/safety_test.go TestRowExpiredAndCitationIntegrity + TestRowUnauthorizedCitation + builder 终检 |
| 19 | 并发 Answer/Health/Reload | matrix/mixed_test.go TestRowMixedConcurrencyAndMetrics + orchestrator 并发基线（-race） |
