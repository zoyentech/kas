# GoldenSet 验收测试

## 1. 设计

GoldenSet 用于可重复比较旧链与 KAS；不以一次模型 judge 作为验收。首轮建议至少 100 条去标识化样本，覆盖常见问法、改写、无答案、过期知识、权限隔离、provider 故障和配置并发切换。P0 安全集分开统计。

每例包含：case_id、risk、question、principal scope、knowledge namespace、corpus generation、valid_at、expected action、expected evidence IDs、forbidden evidence IDs、expected answer facts、must_abstain、tags。样本不含真实姓名、联系方式、token 等无关个人数据。

~~~json
{
  "case_id": "golden-0001",
  "risk": "p0",
  "question": "服务支持哪些配置方式？",
  "principal_scope": "opaque-scope-a",
  "knowledge_namespace": "support-public",
  "corpus_generation": "corpus-2026-10",
  "valid_at": "2026-10-23T00:00:00Z",
  "expected_action": "answer",
  "expected_evidence_ids": ["doc-10#seg-2"],
  "forbidden_evidence_ids": ["doc-obsolete#seg-1"],
  "expected_answer_facts": ["支持配置文件和环境变量"],
  "must_abstain": false,
  "tags": ["freshness", "citation"]
}
~~~

## 2. 验收矩阵

| 场景 | 预期 |
|---|---|
| 标准可答、问法改写 | Evidence recall 达标，事实由引用支持 |
| 空结果/低相关/no-answer | clarify 或 abstain，不编造 |
| Evidence 已过期/尚未生效 | 被过滤，不用于 answer 或 cache |
| 有效期未知且无来源策略 | fail-closed |
| 已批准来源 max-age | 截止时间推导正确，cache TTL 不晚于截止时间 |
| 同问不同 tenant/scope/authorization epoch | 跨 scope cache hit 为 0 |
| corpus generation 改变 | 不读取上一代 cache |
| Reload 并发 | 在途 response/trace 为旧版，新请求为新版 |
| Reload 配置无效 | 返回错误，当前 config_version 不变 |
| Jev 合法响应 | action/目标组属于候选 |
| Jev timeout/5xx | 按规则→cache→abstain 降级，不超 root deadline |
| Jev 非法 action/越界 ID/NaN | 拒绝，记录 contract error 并降级 |
| 多条规则冲突 | 优先级稳定、trace 可解释 |
| 决策缓存命中 | 候选/版本/scope 匹配且 action 重新校验 |
| provider 阻塞 | ctx cancellation 传播；不启动越过 root deadline 的阶段 |
| Health dependency 故障 | required 为 not_ready，optional Jev 为 degraded |
| Metrics 成功/超时/cache/fallback | 增量正确，无敏感/高基数 label |
| 引用完整性 | 所有 citation 属于最终有效 Evidence |
| 并发 Answer/Health/Reload | 不发生 snapshot 混用或缓存串线 |

## 3. 建议通过门槛

以下均为建议目标，冻结前需业务负责人确认：

- P0 安全集：过期 Evidence 用于答案 = 0；跨权限/租户/namespace cache hit = 0；无权限或不存在 citation = 0。
- Evidence Recall@5：全集 ≥0.90，P0 子集 ≥0.95。
- 引用事实支持率 ≥0.95。
- 人工盲评事实正确率 ≥0.90；模型 judge 只辅助排序，不独立裁定。
- 不可答集 abstain/clarify 准确率 ≥0.95。
- 所有 provider 收到可取消 context；请求不超过较早 caller/root deadline（测量误差上限建议 50ms）。
- 配置 Reload 前后版本一致性 100%；cache 命中率为观察值而非质量门槛。
- 每个降级/timeout 都能由 Health、metrics、trace 的安全 reason 和版本定位。

## 4. 对比与验收产物

旧链与 KAS 固定同一语料/索引代次、授权上下文和 valid_at。报告 Evidence Recall@k/Precision@k/MRR/nDCG（有排名标注时）、回答正确/漏答/无据陈述、引用覆盖/准确/过期率、p50/p95/p99、stage timeout、cache hit、Jev fallback 比例。P0 安全集单独报告，不得只给总体均值。

每轮保留 GoldenSet 版本、样本数、corpus/config/model version、旧链版本、时间窗口、指标、人工裁定和缺陷项。未过门槛时旧链继续对用户返回；KAS 只做影子比较。
