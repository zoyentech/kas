// Package internal 提供 GoldenSet 加载、校验、语料构建与评分
// （acceptance 框架内部实现；不进入 SDK 运行时）。
package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Case 是 GoldenSet 用例（docs/design/05 §1 字段）。
type Case struct {
	CaseID               string    `json:"case_id"`
	Risk                 string    `json:"risk"`
	Question             string    `json:"question"`
	PrincipalScope       string    `json:"principal_scope"`
	KnowledgeNamespace   string    `json:"knowledge_namespace"`
	CorpusGeneration     string    `json:"corpus_generation"`
	ValidAt              time.Time `json:"valid_at"`
	ExpectedAction       string    `json:"expected_action"`
	ExpectedEvidenceIDs  []string  `json:"expected_evidence_ids"`
	ForbiddenEvidenceIDs []string  `json:"forbidden_evidence_ids"`
	ExpectedAnswerFacts  []string  `json:"expected_answer_facts"`
	MustAbstain          bool      `json:"must_abstain"`
	Tags                 []string  `json:"tags"`
}

// 场景分类（docs/05 §1：七类，样本配额 ≥8/类）。
var knownTags = map[string]bool{
	"common": true, "paraphrase": true, "no_answer": true, "freshness": true,
	"permission": true, "provider_fault": true, "config_switch": true,
}

var caseIDPattern = regexp.MustCompile(`^golden-[0-9]{4,}$`)

// LoadDir 加载目录下全部 *.json 用例文件并逐例校验（加载期拒绝缺字段/
// 坏格式/未知 risk/时间格式/ID 格式/非法 tag）。
func LoadDir(dir string) ([]Case, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("goldenset: 目录 %s 无用例文件", dir)
	}
	var all []Case
	seen := map[string]bool{}
	for _, path := range entries {
		raw, err := os.ReadFile(path) // #nosec G304 -- 验收框架固定目录
		if err != nil {
			return nil, err
		}
		var batch []Case
		if err := json.Unmarshal(raw, &batch); err != nil {
			return nil, fmt.Errorf("goldenset: %s 解析失败: %w", path, err)
		}
		for i, c := range batch {
			if err := validate(c, path, i); err != nil {
				return nil, err
			}
			if seen[c.CaseID] {
				return nil, fmt.Errorf("goldenset: case_id 重复 %s", c.CaseID)
			}
			seen[c.CaseID] = true
			all = append(all, c)
		}
	}
	return all, nil
}

func validate(c Case, path string, idx int) error {
	at := fmt.Sprintf("%s[%d]", filepath.Base(path), idx)
	if !caseIDPattern.MatchString(c.CaseID) {
		return fmt.Errorf("goldenset %s: case_id %q 不匹配 golden-NNNN 格式", at, c.CaseID)
	}
	if c.Risk != "p0" && c.Risk != "p1" && c.Risk != "p2" {
		return fmt.Errorf("goldenset %s: risk %q 非法", at, c.Risk)
	}
	if strings.TrimSpace(c.Question) == "" {
		return fmt.Errorf("goldenset %s: question 为空", at)
	}
	if c.PrincipalScope == "" || c.KnowledgeNamespace == "" || c.CorpusGeneration == "" {
		return fmt.Errorf("goldenset %s: scope/namespace/generation 缺失", at)
	}
	if c.ValidAt.IsZero() {
		return fmt.Errorf("goldenset %s: valid_at 缺失或格式错误（RFC3339）", at)
	}
	if c.ExpectedAction != "answer" && c.ExpectedAction != "ask_clarification" && c.ExpectedAction != "abstain" {
		return fmt.Errorf("goldenset %s: expected_action %q 非法", at, c.ExpectedAction)
	}
	for _, tag := range c.Tags {
		if !knownTags[tag] {
			return fmt.Errorf("goldenset %s: 未知 tag %q（已知: common/paraphrase/no_answer/freshness/permission/provider_fault/config_switch）", at, tag)
		}
	}
	if c.MustAbstain && len(c.ExpectedEvidenceIDs) > 0 {
		return fmt.Errorf("goldenset %s: must_abstain 与 expected evidence 互斥", at)
	}
	return nil
}

// P0Subset 抽取 risk=p0 子集。
func P0Subset(cases []Case) []Case {
	var out []Case
	for _, c := range cases {
		if c.Risk == "p0" {
			out = append(out, c)
		}
	}
	return out
}

// ContentHash 计算用例集内容摘要（P0 子集冻结与变更检测）。
func ContentHash(cases []Case) (string, error) {
	b, err := json.Marshal(cases)
	if err != nil {
		return "", err
	}
	return digest(b), nil
}
