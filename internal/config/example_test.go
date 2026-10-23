package config

import (
	"reflect"
	"testing"
	"time"
)

// TestExampleYAMLFieldMapping 固化 docs/02 示例 → Config 结构的逐字段映射
// （Story 2.1.1.1 T1：非严格解析后与预期结构逐字段相等）。
func TestExampleYAMLFieldMapping(t *testing.T) {
	got := mustLoadExample(t)

	want := Config{
		SchemaVersion: 1,
		Timeouts: TimeoutBudget{
			Total: D(4800 * time.Millisecond), Preliminary: D(400 * time.Millisecond),
			FirstRound: D(1800 * time.Millisecond), Rejudge: D(900 * time.Millisecond),
			Supplement: D(1100 * time.Millisecond), Reserve: D(600 * time.Millisecond),
		},
		Evidence: EvidenceConfig{
			Provider: "host-search", Namespace: "support-public",
			TopK: 8, MaxCandidateGroups: 4,
			Validity: ValidityConfig{
				RequireValidityPolicyForEachSource: true,
				DefaultMaxAge: map[string]Duration{
					"product_manual": D(720 * time.Hour),
					"service_notice": D(24 * time.Hour),
				},
				ExcludeFutureDated:          true,
				FailClosedOnUnknownValidity: true,
			},
			Cache: EvidenceCacheConfig{
				Enabled: true, Implementation: "memory",
				MaxEntries: 2000, MaxBytes: 33554432, MaxItemBytes: 32768,
				TTL:                 D(120 * time.Second),
				CacheEmptyResults:   false,
				CacheProviderErrors: false,
				Include: []string{
					"normalized_question", "config_version", "knowledge_namespace",
					"corpus_generation", "principal_scope_fingerprint", "authorization_epoch",
				},
			},
		},
		Decision: DecisionConfig{
			Provider: "jev",
			Jev: JevConfig{
				EndpointEnv: "KAS_JEV_ENDPOINT", APIKeyEnv: "KAS_JEV_API_KEY",
				Model: "jev-latest", ModelVersion: "stable", MaxAttempts: 1,
			},
			Fallback: FallbackConfig{
				Order: []string{"jev", "deterministic_rules", "decision_cache", "abstain"},
				Rules: []RuleConfig{
					{Name: "no_valid_evidence", When: "valid_evidence_count_equals_zero", Action: "abstain"},
					{Name: "low_confidence", When: "decision_confidence_below_threshold", Action: "ask_clarification"},
				},
				Cache: DecisionCacheConfig{
					Enabled: true, TTL: D(30 * time.Second), MaxEntries: 500,
					IncludeConfigVersion:                   true,
					IncludeScopeFingerprint:                true,
					IncludeCandidateFingerprint:            true,
					ValidateActionAgainstCurrentCandidates: true,
				},
			},
		},
		Answer: AnswerConfig{
			MaxCitations:                    5,
			RequireCitationForFactualClaims: true,
			AllowAnswerWithoutEvidence:      false,
			OutputOnUnknown:                 "abstain",
		},
		Health: HealthConfig{
			RequiredComponents:                      []string{"evidence_provider"},
			OptionalComponents:                      []string{"jev"},
			CheckTimeout:                            D(250 * time.Millisecond),
			DegradedWhenOptionalProviderUnavailable: true,
		},
		Metrics: MetricsConfig{
			Enabled: true, Namespace: "kas",
			LabelAllowlist: []string{"operation", "stage", "provider", "outcome", "action", "cache_result", "fallback_reason"},
		},
		Tracing: TracingConfig{
			Enabled: true, CaptureQuestionText: false, CaptureEvidenceText: false,
			Include: []string{"config_version", "evidence_provider", "decision_provider", "model", "model_version", "fallback_route", "cache_result"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("docs/02 示例字段映射不符\n got: %#v\nwant: %#v", got, want)
	}
}
