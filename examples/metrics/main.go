// Package main 是宿主接入监控的可运行演示（Story 8.7.1.3；docs/monitoring.md）。
//
//	go run ./examples/metrics
//
// 行为：本进程内起 fake Jev（decide+balance）→ 构造 KAS Client（决策链/熔断/
// 余额监测全开）→ 127.0.0.1:8080 暴露 /metrics（promhttp）→ 后台每秒混合流量
// （正常/无证据/偶发空问题）且余额逐步递减演示 normal→low→critical 迁移。
// 配套：cd monitoring && docker compose up -d → Grafana :3000 即见出数。
package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	kas "github.com/zoyentech/kas"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence/inmem"
)

// demoYAML 演示配置：余额监测 + 熔断默认开 + 全指标 label。
const demoYAML = `
schema_version: 1
timeouts: {total: 4800ms, preliminary: 400ms, first_round: 1800ms, rejudge: 900ms, supplement: 1100ms, reserve: 600ms}
evidence:
  provider: inmem
  namespace: demo
  top_k: 5
  max_candidate_groups: 2
  validity: {require_validity_policy_for_each_source: false, default_max_age: {doc-1: 720h}, fail_closed_on_unknown_validity: true}
  cache: {enabled: true, implementation: memory, max_entries: 100, max_bytes: 8388608, max_item_bytes: 1048576, ttl: 60s, include: [normalized_question, config_version, principal_scope_fingerprint]}
decision:
  provider: jev
  jev:
    endpoint_env: KAS_DEMO_JEV_ENDPOINT
    api_key_env: KAS_DEMO_JEV_API_KEY
    balance_endpoint_env: KAS_DEMO_JEV_BALANCE
    model: fake-jev
    model_version: v1
    max_attempts: 1
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics:
  enabled: true
  namespace: kas
  label_allowlist: [operation, outcome, action, stage, provider, result, reason, route, component, status, deadline_stage, error_class, source, unit]
tracing: {enabled: false}
balance:
  enabled: true
  interval: 5s
  low_threshold: 300
  critical_threshold: 50
`

func main() {
	// fake Jev（演示用；生产替换为真实 endpoint env 注入）
	var balance atomic.Int64 // 余额 ×100 存储（演示递减）
	balance.Store(50000)     // 500.00 credit
	fake := fakeJev(&balance)
	_ = fake // httptest server URL 在闭包内引用

	for k, v := range map[string]string{
		"KAS_DEMO_JEV_ENDPOINT": fake + "/decide",
		"KAS_DEMO_JEV_API_KEY":  "demo-key",
		"KAS_DEMO_JEV_BALANCE":  fake + "/balance",
	} {
		if err := os.Setenv(k, v); err != nil {
			log.Fatalf("设置演示 env %s 失败: %v", k, err)
		}
	}

	cfg, err := kas.LoadConfig(strings.NewReader(demoYAML))
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	until := time.Now().Add(24 * time.Hour)
	client, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider: inmem.New([]domain.Evidence{
			{
				ID: "doc-1#seg-1", SourceID: "doc-1", Title: "配置方式",
				Excerpt: "支持配置文件与环境变量", ValidUntil: &until, ContentVersion: "v1", Score: 0.9,
			},
		}),
	}, kas.WithPrometheusRegisterer(prometheus.DefaultRegisterer))
	if err != nil {
		log.Fatalf("New 失败: %v", err)
	}
	defer func() { _ = client.Close() }()

	// 流量发生器：每秒一请求（正常/无证据交替，偶发空问题制造 invalid_request）
	questions := []string{"服务支持哪些配置方式？", "不存在的冷门问题？", "服务支持哪些配置方式？", ""}
	go func() {
		for i := 0; ; i++ {
			_, _ = client.Answer(context.Background(), kas.Request{
				Question:  questions[i%len(questions)],
				Principal: kas.PrincipalScope{TenantKey: "demo", ScopeFingerprint: "fp-demo", AuthorizationEpoch: "e1"},
				Knowledge: kas.KnowledgeScope{Namespace: "demo", CorpusGeneration: "g1"},
			})
			balance.Add(-137) // 每请求扣 1.37 credit（演示余额递减）
			time.Sleep(time.Second)
		}
	}()

	// 周期 Health 探测（10s）：真实宿主运维形态，填充 kas_health_status 面板
	go func() {
		for {
			_ = client.Health(context.Background())
			time.Sleep(10 * time.Second)
		}
	}()

	// /metrics：宿主用 promhttp 暴露（监控栈抓取入口）
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: "127.0.0.1:8080", Handler: mux}
	go func() {
		log.Println("metrics demo: http://127.0.0.1:8080/metrics（monitoring/prometheus.yml 抓取目标）")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("metrics server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// fakeJev 进程内 decide + balance 端点。
func fakeJev(balance *atomic.Int64) string {
	mux := http.NewServeMux()
	mux.HandleFunc("/decide", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"action":"answer","target_group_ids":["g1"],"confidence":0.9,` +
			`"reason_code":"ok","model":"fake-jev","model_version":"v1","answer_text":"根据知识库整理的回答。"}`))
	})
	mux.HandleFunc("/balance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		amount := strconv.FormatFloat(float64(balance.Load())/100, 'f', 2, 64)
		_, _ = w.Write([]byte(`{"amount":` + amount + `,"unit":"credit"}`))
	})
	return httptest.NewServer(mux).URL
}
