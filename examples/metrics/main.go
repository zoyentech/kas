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
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
  sources: # 各源独立阈值（v1.4 多源）
    deepseek: {low_threshold: 100, critical_threshold: 10}
    aliyun: {low_threshold: 200}
`

func main() {
	// fake 组件端点（演示用；生产替换为真实 endpoint env 注入）。
	// 余额为"趋势函数"：锚定持久化时间基准（/tmp/kas-demo-epoch，首启写入），
	// 按墙钟线性递减、触地板自动充值回满——重启不重置（无回跳）、不穿底，
	// 周期性走完 normal→low→critical→充值 全状态。
	epoch := demoEpoch()
	trend := newBalanceTrends(epoch)
	fake := fakeJev(trend.jev)
	fakes := fakeComponentEndpoints(trend.deepseek, trend.aliyun)

	// 模式：KAS_DEMO_FAKE=0 走真实凭据（DEEPSEEK_TOKEN/ALIBABA_CLOUD_*/QWEN_APIKEY
	// 由进程环境提供，例如 ~/.config/ai-support-platform/ai.env 注入）；默认 fake。
	realMode := os.Getenv("KAS_DEMO_FAKE") == "0"
	envs := map[string]string{
		"KAS_DEMO_JEV_ENDPOINT": fake + "/decide", // Jev 无官方余额接口（O-2），
		"KAS_DEMO_JEV_API_KEY":  "demo-key",       // decide/balance 始终用 fake
		"KAS_DEMO_JEV_BALANCE":  fake + "/balance",
	}
	if realMode {
		for _, k := range []string{"DEEPSEEK_TOKEN", "ALIBABA_CLOUD_ACCESS_KEY_ID", "ALIBABA_CLOUD_ACCESS_KEY_SECRET"} {
			if os.Getenv(k) == "" {
				log.Fatalf("KAS_DEMO_FAKE=0 需要真实凭据 env：%s 未设置", k)
			}
		}
		log.Println("余额模式=真实：deepseek→api.deepseek.com，aliyun/qwen→BSS QueryAccountBalance；Jev 仍为 fake（O-2）")
	} else {
		envs["DEEPSEEK_TOKEN"] = "demo-ds-key"
		envs["DEEPSEEK_BALANCE_ENDPOINT"] = fakes.ds + "/user/balance"
		envs["ALIBABA_CLOUD_ACCESS_KEY_ID"] = "demo-ak"
		envs["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] = "demo-sk"
		envs["ALIBABA_CLOUD_BSS_ENDPOINT"] = fakes.aliyun
		envs["QWEN_APIKEY"] = "demo-qwen-key"
		log.Println("余额模式=模拟（KAS_DEMO_FAKE=0 切真实凭据）")
	}
	for k, v := range envs {
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
		log.Println("余额源演示: jev(credit 递减) + deepseek/aliyun/qwen(CNY)；看板按 source 分列")
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

// fakeEndpoints 聚合两个演示组件端点。
type fakeEndpoints struct{ ds, aliyun string }

// fakeComponentEndpoints DeepSeek（/user/balance 契约）与阿里云 BSS
// （QueryAccountBalance 响应形态；签名校验由真实端点负责，fake 忽略）本地端点。
func fakeComponentEndpoints(dsBalance, aliyunBalance trendFn) fakeEndpoints {
	dsMux := http.NewServeMux()
	dsMux.HandleFunc("/user/balance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		amount := strconv.FormatFloat(dsBalance(), 'f', 2, 64)
		_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[` +
			`{"currency":"CNY","total_balance":"` + amount + `","granted_balance":"0.00","topped_up_balance":"` + amount + `"}]}`))
	})
	dsSrv := httptest.NewServer(dsMux)

	aliMux := http.NewServeMux()
	aliMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		amount := strconv.FormatFloat(aliyunBalance(), 'f', 2, 64)
		_, _ = w.Write([]byte(`{"Success":true,"Data":{"AvailableCashAmount":"` + amount + `","Currency":"CNY"}}`))
	})
	aliSrv := httptest.NewServer(aliMux)
	return fakeEndpoints{ds: dsSrv.URL, aliyun: aliSrv.URL}
}

// trendFn 返回当前时刻的模拟余额（元单位；时间锚定，重启续走）。
type trendFn func() float64

// balanceTrend 三个组件的余额趋势参数（full→floor 线性递减，触底回满）。
type balanceTrends struct{ jev, deepseek, aliyun trendFn }

// newBalanceTrends 各组件周期：jev≈30min、deepseek≈45min、aliyun≈60min
// （错峰触发 low/critical，便于看板观察状态迁移与充值回跳）。
func newBalanceTrends(epoch time.Time) *balanceTrends {
	return &balanceTrends{
		jev:      sawtooth(epoch, 500, 25, 30*time.Minute),
		deepseek: sawtooth(epoch, 120, 5, 45*time.Minute),
		aliyun:   sawtooth(epoch, 666.66, 25, 60*time.Minute),
	}
}

// sawtooth 返回 f(t)：从 full 线性降至 floor 后回满，周期 period（纯时间函数，
// 幂等、跨进程一致）。
func sawtooth(epoch time.Time, full, floor float64, period time.Duration) trendFn {
	drop := full - floor
	rate := drop / period.Seconds() // 单位/秒
	return func() float64 {
		elapsed := time.Since(epoch).Seconds()
		inCycle := math.Mod(elapsed, period.Seconds())
		v := full - rate*inCycle
		if v < floor {
			return floor
		}
		return v
	}
}

// demoEpoch 演示时间基准：/tmp/kas-demo-epoch 首启写入、后续复用（重启续走）。
func demoEpoch() time.Time {
	const path = "/tmp/kas-demo-epoch"
	if b, err := os.ReadFile(path); err == nil {
		if ts, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil {
			return time.Unix(ts, 0)
		}
	}
	now := time.Now()
	_ = os.WriteFile(path, []byte(strconv.FormatInt(now.Unix(), 10)), 0o644)
	return now
}

// fakeJev 进程内 decide + balance 端点。
func fakeJev(balance trendFn) string {
	mux := http.NewServeMux()
	mux.HandleFunc("/decide", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"action":"answer","target_group_ids":["g1"],"confidence":0.9,` +
			`"reason_code":"ok","model":"fake-jev","model_version":"v1","answer_text":"根据知识库整理的回答。"}`))
	})
	mux.HandleFunc("/balance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		amount := strconv.FormatFloat(balance(), 'f', 2, 64)
		_, _ = w.Write([]byte(`{"amount":` + amount + `,"unit":"credit"}`))
	})
	return httptest.NewServer(mux).URL
}
