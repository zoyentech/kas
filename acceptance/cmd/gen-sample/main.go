// gen-sample 生成去标识化 GoldenSet 合成样本（Story 7.1.1.1 T2）。
// O-6 真实语料审批前以合成样本占位：覆盖七类场景、每类 ≥8 条、合计 ≥100 条。
// 用法：go run ./acceptance/cmd/gen-sample -out acceptance/goldenset/cases
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	out := flag.String("out", "acceptance/goldenset/cases", "输出目录")
	flag.Parse()
	validAt := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)

	type batch struct {
		file  string
		tag   string
		risk  string
		count int
		build func(i int, tag string) map[string]interface{}
	}

	batches := []batch{
		{"common.json", "common", "p2", 24, func(i int, tag string) map[string]interface{} {
			id := fmt.Sprintf("golden-1%03d", i)
			return map[string]interface{}{
				"case_id": id, "risk": "p2", "question": fmt.Sprintf("产品手册第 %d 章支持哪些配置方式？", i),
				"principal_scope": "scope-common", "knowledge_namespace": "support-public",
				"corpus_generation": "corpus-2026-10", "valid_at": validAt.Format(time.RFC3339),
				"expected_action":        "answer",
				"expected_evidence_ids":  []string{fmt.Sprintf("doc-%d#seg-%d", 100+i, 2)},
				"forbidden_evidence_ids": []string{fmt.Sprintf("doc-obsolete-%d#seg-1", i)},
				"expected_answer_facts":  []string{fmt.Sprintf("支持配置文件 %d 号方案与环境变量", i)},
				"must_abstain":           false, "tags": []string{tag},
			}
		}},
		{"paraphrase.json", "paraphrase", "p2", 16, func(i int, tag string) map[string]interface{} {
			forms := []string{
				"怎么给服务设置超时？", "如何配置服务的超时时间？", "超时时间在哪里设置？",
				"服务超时如何调整？", "设置超时的方法是什么？", "请问超时怎么配？",
				"超时参数怎么改？", "哪里修改服务超时？", "调整超时时间的步骤？",
				"超时配置的入口？", "改变超时怎么做？", "服务超时设定方法？",
			}
			id := fmt.Sprintf("golden-2%03d", i)
			return map[string]interface{}{
				"case_id": id, "risk": "p2", "question": forms[i%len(forms)],
				"principal_scope": "scope-common", "knowledge_namespace": "support-public",
				"corpus_generation": "corpus-2026-10", "valid_at": validAt.Format(time.RFC3339),
				"expected_action":       "answer",
				"expected_evidence_ids": []string{"doc-200#seg-1"},
				"expected_answer_facts": []string{"在配置文件的 timeouts 节设置"},
				"must_abstain":          false, "tags": []string{tag},
			}
		}},
		{"no_answer.json", "no_answer", "p1", 12, func(i int, tag string) map[string]interface{} {
			questions := []string{
				"服务支持火星部署吗？", "能让服务回到过去吗？", "支持用意念配置吗？",
				"服务能煮咖啡吗？", "支持隐身模式吗？", "能把超时设成昨天吗？",
				"支持用摩斯电码提问吗？", "服务会读心术吗？", "支持超光速响应吗？",
				"能让答案自动变成对的吗？",
			}
			id := fmt.Sprintf("golden-3%03d", i)
			return map[string]interface{}{
				"case_id": id, "risk": "p1", "question": questions[i%len(questions)],
				"principal_scope": "scope-common", "knowledge_namespace": "support-public",
				"corpus_generation": "corpus-2026-10", "valid_at": validAt.Format(time.RFC3339),
				"expected_action":       "abstain",
				"expected_answer_facts": []string{},
				"must_abstain":          true, "tags": []string{tag},
			}
		}},
		{"freshness.json", "freshness", "p0", 16, func(i int, tag string) map[string]interface{} {
			id := fmt.Sprintf("golden-4%03d", i)
			return map[string]interface{}{
				"case_id": id, "risk": "p0", "question": fmt.Sprintf("第 %d 号公告的当前有效内容是什么？", i),
				"principal_scope": "scope-common", "knowledge_namespace": "support-public",
				"corpus_generation": "corpus-2026-10", "valid_at": validAt.Format(time.RFC3339),
				"expected_action":        "answer",
				"expected_evidence_ids":  []string{fmt.Sprintf("notice-%d#seg-1", i)},
				"forbidden_evidence_ids": []string{fmt.Sprintf("notice-expired-%d#seg-1", i)},
				"expected_answer_facts":  []string{fmt.Sprintf("第 %d 号公告当前版本内容", i)},
				"must_abstain":           false, "tags": []string{tag, "freshness"},
			}
		}},
		{"permission.json", "permission", "p0", 14, func(i int, tag string) map[string]interface{} {
			id := fmt.Sprintf("golden-5%03d", i)
			scope := "scope-internal"
			if i%2 == 0 {
				scope = "scope-guest"
			}
			return map[string]interface{}{
				"case_id": id, "risk": "p0", "question": fmt.Sprintf("内部知识库第 %d 条的内容？", i),
				"principal_scope": scope, "knowledge_namespace": "support-public",
				"corpus_generation": "corpus-2026-10", "valid_at": validAt.Format(time.RFC3339),
				"expected_action": func() string {
					if i%2 == 0 {
						return "abstain" // guest 无权
					}
					return "answer"
				}(),
				"expected_evidence_ids": func() []string {
					if i%2 == 0 {
						return nil
					}
					return []string{fmt.Sprintf("kb-internal-%d#seg-1", i)}
				}(),
				"forbidden_evidence_ids": func() []string {
					if i%2 == 0 {
						return []string{fmt.Sprintf("kb-internal-%d#seg-1", i)}
					}
					// 有权用例的 forbidden 是另一份越权锁定证据
					return []string{fmt.Sprintf("kb-locked-%d#seg-1", i)}
				}(),
				"expected_answer_facts": []string{},
				"must_abstain":          i%2 == 0,
				"tags":                  []string{tag, "permission"},
			}
		}},
		{"provider_fault.json", "provider_fault", "p1", 10, func(i int, tag string) map[string]interface{} {
			id := fmt.Sprintf("golden-6%03d", i)
			return map[string]interface{}{
				"case_id": id, "risk": "p1", "question": fmt.Sprintf("故障演练问题 %d：服务的降级行为？", i),
				"principal_scope": "scope-common", "knowledge_namespace": "support-public",
				"corpus_generation": "corpus-2026-10", "valid_at": validAt.Format(time.RFC3339),
				"expected_action":       "answer",
				"expected_evidence_ids": []string{fmt.Sprintf("doc-runbook-%d#seg-3", i)},
				"expected_answer_facts": []string{"按降级链处理"},
				"must_abstain":          false, "tags": []string{tag},
			}
		}},
		{"config_switch.json", "config_switch", "p1", 10, func(i int, tag string) map[string]interface{} {
			id := fmt.Sprintf("golden-7%03d", i)
			gen := "corpus-2026-10"
			if i%2 == 1 {
				gen = "corpus-2026-11"
			}
			return map[string]interface{}{
				"case_id": id, "risk": "p1", "question": fmt.Sprintf("配置切换演练问题 %d", i),
				"principal_scope": "scope-common", "knowledge_namespace": "support-public",
				"corpus_generation": gen, "valid_at": validAt.Format(time.RFC3339),
				"expected_action":       "answer",
				"expected_evidence_ids": []string{fmt.Sprintf("doc-cfg-%d#seg-1", i)},
				"expected_answer_facts": []string{"配置切换后行为一致"},
				"must_abstain":          false, "tags": []string{tag},
			}
		}},
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	total := 0
	var p0 int
	for _, b := range batches {
		var arr []map[string]interface{}
		for i := 0; i < b.count; i++ {
			c := b.build(i, b.tag)
			arr = append(arr, c)
			total++
			if c["risk"] == "p0" {
				p0++
			}
		}
		raw, err := json.MarshalIndent(arr, "", "  ")
		if err != nil {
			fail(err)
		}
		path := filepath.Join(*out, b.file)
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("%s: %d 条\n", path, len(arr))
	}
	fmt.Printf("合计 %d 条（P0 %d 条）\n", total, p0)
	if total < 100 {
		fail(fmt.Errorf("样本不足 100 条"))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gen-sample:", err)
	os.Exit(1)
}
