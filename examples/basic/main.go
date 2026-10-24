// Package main 是 KAS 最小嵌入示例（README 三步嵌入的可运行版本；
// 使用 inmem provider 与 go run 可直接跑通）。
//
//	go run ./examples/basic
//
// 预期输出：action=answer 引用 2 条 config_version=xxxx
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence/inmem"
)

// stubDecision 演示宿主自定义决策器（作为降级链首选级注入）。
type stubDecision struct{}

func (stubDecision) Decide(ctx context.Context, in kas.DecisionInput) (kas.Decision, error) {
	if len(in.Groups) == 0 || len(in.Groups[0].Evidence) == 0 {
		return kas.Decision{Action: kas.ActionAbstain, Provider: "stub"}, nil
	}
	return kas.Decision{
		Action: kas.ActionAnswer, TargetGroupIDs: []string{in.Groups[0].ID},
		Confidence: 0.9, Provider: "stub", AnswerText: "根据知识库整理的回答。",
	}, nil
}

func (stubDecision) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

func main() {
	// 1) Load：YAML 严格解析 + 全部启动校验
	cfg, err := kas.LoadConfigFile("docs/design/02-example.yaml")
	if err != nil {
		// 演示环境无 env 注入时降级用最小配置
		cfg, err = kas.LoadConfig(strings.NewReader(minimalYAML))
		if err != nil {
			panic(err)
		}
	}

	// 2) New：装配 Client（缓存/降级链/护栏自动接线）
	until := time.Now().Add(24 * time.Hour)
	client, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider: inmem.New([]domain.Evidence{
			{ID: "doc-10#seg-2", SourceID: "doc-10", Title: "配置方式",
				Excerpt: "支持配置文件与环境变量", ValidUntil: &until, ContentVersion: "v3"},
		}),
		DecisionProvider: stubDecision{},
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = client.Close() }()

	// 3) Answer / Health
	resp, err := client.Answer(context.Background(), kas.Request{
		Question:  "服务支持哪些配置方式？",
		Principal: kas.PrincipalScope{TenantKey: "demo", ScopeFingerprint: "fp-1", AuthorizationEpoch: "e1"},
		Knowledge: kas.KnowledgeScope{Namespace: cfg.Evidence.Namespace, CorpusGeneration: "g1"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("action=%s 引用 %d 条 config_version=%s\n", resp.Action, len(resp.Citations), resp.ConfigVersion)

	health := client.Health(context.Background())
	fmt.Printf("health=%s\n", health.Status)
}

const minimalYAML = `
schema_version: 1
timeouts: {total: 4800ms, preliminary: 400ms, first_round: 1800ms, rejudge: 900ms, supplement: 1100ms, reserve: 600ms}
evidence:
  provider: inmem
  namespace: demo
  top_k: 5
  max_candidate_groups: 2
  validity: {require_validity_policy_for_each_source: false, default_max_age: {doc-10: 720h}, fail_closed_on_unknown_validity: true}
  cache: {enabled: true, implementation: memory, max_entries: 100, max_bytes: 8388608, max_item_bytes: 1048576, ttl: 60s, include: [normalized_question, config_version, principal_scope_fingerprint]}
decision:
  provider: stub
  jev: {endpoint_env: KAS_JEV_ENDPOINT, api_key_env: KAS_JEV_API_KEY, model: demo, model_version: v1, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
`
