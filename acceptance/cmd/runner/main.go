// runner 是 GoldenSet 验收执行器（Story 7.1.1.2）：
// 固定 corpus generation/授权上下文/valid_at 锚点跑全部用例，输出 JSONL
// 原始结果与 Markdown/JSON 双格式报告，按 docs/05 §3 建议门槛判定。
// 用法：go run ./acceptance/cmd/runner [-cases dir] [-out dir]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	kas "github.com/zoyentech/kas"
	acc "github.com/zoyentech/kas/acceptance/internal"
)

// 门槛（docs/05 §3 建议值；O-3 冻结前按建议值执行，冻结后替换）。
const (
	thresholdRecallAll   = 0.90
	thresholdRecallP0    = 0.95
	thresholdFactSupport = 0.95
	thresholdAbstain     = 0.95
)

const runnerYAML = `
schema_version: 1
timeouts: {total: 4800ms, preliminary: 400ms, first_round: 1800ms, rejudge: 900ms, supplement: 1100ms, reserve: 600ms}
evidence:
  provider: goldenset
  namespace: support-public
  top_k: 8
  max_candidate_groups: 4
  validity:
    require_validity_policy_for_each_source: false
    default_max_age: {doc: 720h, notice: 24h, kb-internal: 720h}
    exclude_future_dated: true
    fail_closed_on_unknown_validity: true
  cache:
    enabled: true
    implementation: memory
    max_entries: 2000
    max_bytes: 33554432
    max_item_bytes: 32768
    ttl: 120s
    include: [normalized_question, config_version, principal_scope_fingerprint, knowledge_namespace, corpus_generation, authorization_epoch]
decision:
  provider: gold
  jev: {endpoint_env: KAS_GOLD_JEV_ENDPOINT, api_key_env: KAS_GOLD_JEV_API_KEY, model: stub, model_version: v1, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: no_valid_evidence, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
`

func main() {
	casesDir := flag.String("cases", "acceptance/goldenset/cases", "用例目录")
	outDir := flag.String("out", "acceptance/report", "报告输出目录")
	flag.Parse()

	cases, err := acc.LoadDir(*casesDir)
	if err != nil {
		fail(err)
	}
	// 语料锚点 = 执行时刻（用例 valid_at 是冻结元数据；证据时间相对锚点构造）
	anchor := time.Now()

	cfg, err := kas.LoadConfig(strings.NewReader(runnerYAML))
	if err != nil {
		fail(fmt.Errorf("runner 配置非法: %w", err))
	}
	corpus := acc.BuildCorpus(cases, anchor)
	client, err := kas.New(cfg, kas.Dependencies{
		EvidenceProvider:  corpus,
		DecisionProvider:  acc.GoldDecision{},
		PermissionChecker: kas.NewTagChecker(map[string][]string{"scope-internal": {"kb:internal"}}),
	})
	if err != nil {
		fail(err)
	}
	defer func() { _ = client.Close() }()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(err)
	}
	jsonl, err := os.Create(filepath.Join(*outDir, "results.jsonl")) // #nosec G304 -- 固定输出目录
	if err != nil {
		fail(err)
	}
	defer func() { _ = jsonl.Close() }()

	type runRecord struct {
		CaseID    string   `json:"case_id"`
		Action    string   `json:"action"`
		Citations []string `json:"citations"`
		Duration  float64  `json:"duration_ms"`
		Degraded  bool     `json:"degraded"`
		Error     string   `json:"error,omitempty"`
	}

	var scores []acc.CaseScore
	var p0Scores []acc.CaseScore
	var durations []float64
	var records []runRecord
	failed := 0

	for _, c := range cases {
		req := kas.Request{
			Question:  c.Question,
			Principal: kas.PrincipalScope{TenantKey: "acc", ScopeFingerprint: c.PrincipalScope, AuthorizationEpoch: "e1"},
			Knowledge: kas.KnowledgeScope{Namespace: c.KnowledgeNamespace, CorpusGeneration: c.CorpusGeneration},
			RequestID: "acc-" + c.CaseID,
		}
		ctx := acc.WithFacts(context.Background(), acc.JoinFacts(c.ExpectedAnswerFacts))
		start := time.Now()
		resp, err := client.Answer(ctx, req)
		durMS := float64(time.Since(start).Microseconds()) / 1000.0

		var cited []string
		for _, cit := range resp.Citations {
			cited = append(cited, cit.EvidenceID)
		}
		action := string(resp.Action)
		if err != nil {
			action = "error"
			failed++
		}
		s := acc.ScoreCase(acc.ScoreInput{
			Case: c, Action: action, Citations: cited, Answer: resp.Answer, DurationMS: durMS,
		})
		scores = append(scores, s)
		if c.Risk == "p0" {
			p0Scores = append(p0Scores, s)
		}
		durations = append(durations, durMS)
		rec := runRecord{CaseID: c.CaseID, Action: action, Citations: cited, Duration: durMS, Degraded: resp.Degraded}
		if err != nil {
			rec.Error = err.Error()
		}
		records = append(records, rec)
		b, _ := json.Marshal(rec)
		if _, werr := jsonl.WriteString(string(b) + "\n"); werr != nil {
			fail(werr)
		}
	}

	all := acc.Aggregate(scores, durations)
	p0 := acc.Aggregate(p0Scores, nil)
	report := buildReport(cases, all, p0, failed, anchor)

	jsonOut, _ := json.MarshalIndent(map[string]interface{}{
		"goldenset_version": casesHash(cases), "count": len(cases), "failed": failed,
		"all": all, "p0": p0, "thresholds": map[string]float64{
			"recall_all": thresholdRecallAll, "recall_p0": thresholdRecallP0,
			"fact_support": thresholdFactSupport, "abstain": thresholdAbstain,
		},
	}, "", "  ")
	if err := os.WriteFile(filepath.Join(*outDir, "report.json"), jsonOut, 0o644); err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "report.md"), []byte(report), 0o644); err != nil {
		fail(err)
	}
	fmt.Print(report)

	// 门槛判定（失败即非零退出，CI 集成）
	verdict := all.P0Violations == 0 &&
		all.RecallAt5 >= thresholdRecallAll &&
		p0.RecallAt5 >= thresholdRecallP0 &&
		all.FactSupport >= thresholdFactSupport &&
		all.AbstainAccuracy >= thresholdAbstain
	if !verdict {
		fmt.Println("\n门槛判定：FAIL")
		os.Exit(1)
	}
	fmt.Println("\n门槛判定：PASS")
}

func buildReport(cases []acc.Case, all, p0 acc.SetScore, failed int, anchor time.Time) string {
	hash, _ := acc.ContentHash(cases)
	p0Count := 0
	for _, c := range cases {
		if c.Risk == "p0" {
			p0Count++
		}
	}
	byTag := map[string]int{}
	for _, c := range cases {
		for _, t := range c.Tags {
			byTag[t]++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# GoldenSet 验收报告\n\n")
	fmt.Fprintf(&b, "- GoldenSet 版本（内容 hash）: `%s`\n", hash[:16])
	fmt.Fprintf(&b, "- 样本数: %d（P0 子集 %d）\n", len(cases), p0Count)
	fmt.Fprintf(&b, "- 执行时间窗口: %s ~（anchor=%s）\n", time.Now().Format(time.RFC3339), anchor.Format(time.RFC3339))
	fmt.Fprintf(&b, "- corpus/config/model: corpus-2026-10 / runner-v1 / goldenset-stub v1\n")
	fmt.Fprintf(&b, "- 失败用例: %d\n\n", failed)
	fmt.Fprintf(&b, "## 指标（P0 与全集分开输出，docs/05 §4）\n\n")
	fmt.Fprintf(&b, "| 指标 | 全集 | P0 子集 | 门槛 |\n|---|---|---|---|\n")
	fmt.Fprintf(&b, "| Evidence Recall@5 | %.3f | %.3f | 全集≥%.2f / P0≥%.2f |\n", all.RecallAt5, p0.RecallAt5, thresholdRecallAll, thresholdRecallP0)
	fmt.Fprintf(&b, "| Precision@5 | %.3f | %.3f | — |\n", all.PrecisionAt5, p0.PrecisionAt5)
	fmt.Fprintf(&b, "| MRR | %.3f | %.3f | — |\n", all.MRR, p0.MRR)
	fmt.Fprintf(&b, "| nDCG | %.3f | %.3f | — |\n", all.NDCG, p0.NDCG)
	fmt.Fprintf(&b, "| 引用事实支持率 | %.3f | %.3f | ≥%.2f |\n", all.FactSupport, p0.FactSupport, thresholdFactSupport)
	fmt.Fprintf(&b, "| 不可答 abstain/clarify 准确率 | %.3f | %.3f | ≥%.2f |\n", all.AbstainAccuracy, p0.AbstainAccuracy, thresholdAbstain)
	fmt.Fprintf(&b, "| P0 安全违规（过期引用+forbidden 引用+无据回答） | **%d** | — | = 0 |\n", all.P0Violations)
	fmt.Fprintf(&b, "| 时延 p50/p95/p99 (ms) | %.1f / %.1f / %.1f | — | 观察值 |\n\n", all.MedianMS, all.P95MS, all.P99MS)
	fmt.Fprintf(&b, "## 场景覆盖\n\n")
	for tag, n := range byTag {
		fmt.Fprintf(&b, "- %s: %d 条\n", tag, n)
	}
	fmt.Fprintf(&b, "\n> 人工盲评与缺陷项：占位（首轮自动化报告；人工裁定按 docs/05 §4 流程补充）。\n")
	return b.String()
}

func casesHash(cases []acc.Case) string {
	h, _ := acc.ContentHash(cases)
	return h
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "runner:", err)
	os.Exit(1)
}
