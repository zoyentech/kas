package config

import "testing"

// exampleYAMLPath 指向仓库内迁移的设计示例（Stage 2 DoD：docs/02 示例 YAML
// 可被 LoadConfigFile 加载并通过 ValidateConfig）。
const exampleYAMLPath = "../../docs/design/02-example.yaml"

// mustLoadExample 加载 docs/02 示例（测试前置：该文件必须可加载且合法）。
func mustLoadExample(t *testing.T) Config {
	t.Helper()
	cfg, err := LoadConfigFile(exampleYAMLPath)
	if err != nil {
		t.Fatalf("docs/02 示例配置必须可加载并通过校验，实际错误:\n%v", err)
	}
	return cfg
}

// minimalValidYAML 是满足全部校验的最小配置（程序化测试基线）。
const minimalValidYAML = `
schema_version: 1
timeouts:
  total: 4800ms
  preliminary: 400ms
  first_round: 1800ms
  rejudge: 900ms
  supplement: 1100ms
  reserve: 600ms
evidence:
  provider: host-search
  namespace: support-public
  top_k: 8
  max_candidate_groups: 4
  validity:
    require_validity_policy_for_each_source: true
    default_max_age:
      product_manual: 720h
      service_notice: 24h
    exclude_future_dated: true
    fail_closed_on_unknown_validity: true
  cache:
    enabled: true
    implementation: memory
    max_entries: 2000
    max_bytes: 33554432
    max_item_bytes: 32768
    ttl: 120s
    cache_empty_results: false
    cache_provider_errors: false
    include: [normalized_question, config_version, principal_scope_fingerprint, knowledge_namespace, corpus_generation, authorization_epoch]
decision:
  provider: jev
  jev:
    endpoint_env: KAS_JEV_ENDPOINT
    api_key_env: KAS_JEV_API_KEY
    model: jev-latest
    model_version: stable
    max_attempts: 1
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules:
      - name: no_valid_evidence
        when: valid_evidence_count_equals_zero
        action: abstain
    cache:
      enabled: true
      ttl: 30s
      max_entries: 500
      include_config_version: true
      include_scope_fingerprint: true
      include_candidate_fingerprint: true
      validate_action_against_current_candidates: true
answer:
  max_citations: 5
  require_citation_for_factual_claims: true
  allow_answer_without_evidence: false
  output_on_unknown: abstain
health:
  required_components: [evidence_provider]
  optional_components: [jev]
  check_timeout: 250ms
  degraded_when_optional_provider_unavailable: true
metrics:
  enabled: true
  namespace: kas
  label_allowlist: [operation, stage, provider, outcome, action, cache_result, fallback_reason]
tracing:
  enabled: true
  capture_question_text: false
  capture_evidence_text: false
  include: [config_version, evidence_provider, decision_provider, model, model_version, fallback_route, cache_result]
`
