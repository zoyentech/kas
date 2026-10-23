package cache

import (
	"strings"
	"testing"
)

// TestNormalizeBase 基础规范化：NFKC + trim + 空白折叠（Story 4.3.1.1 T1）。
func TestNormalizeBase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"  服务 支持哪些 配置方式？  ", "服务 支持哪些 配置方式?"}, // ？经 NFKC 折叠为 ?
		{"a\t\tb\n\nc", "a b c"},
		{"", ""},
		{"   ", ""},
		{"，vs ,", ",vs ,"}, // 全角逗号经 NFKC 折叠为半角
		{"１２３", "123"},     // 全角数字 NFKC → 半角
		{"ﬁle", "file"},    // 连字 ﬁ NFKC 分解
		{"ＡＢＣ Ｄ", "ABC D"}, // 全角字母 NFKC → 半角
		{"a　b", "a b"},     // 全角空格折叠
	}
	for _, tc := range tests {
		if got := Normalize(tc.in, CaseOff); got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNormalizeCasePolicy 大小写策略（Story 4.3.1.1 T2）。
func TestNormalizeCasePolicy(t *testing.T) {
	// CaseASCII：仅 ASCII 字母小写
	if got := Normalize("iPhone 15 PRO", CaseASCII); got != "iphone 15 pro" {
		t.Fatalf("CaseASCII: %q", got)
	}
	// CaseOff：保持原样
	if got := Normalize("iPhone 15 PRO", CaseOff); got != "iPhone 15 PRO" {
		t.Fatalf("CaseOff: %q", got)
	}
	// 非 ASCII 大小写不受影响：土耳其 İ/ı 与德语 ß 保持
	if got := Normalize("STRASSE straße", CaseASCII); got != "strasse straße" {
		t.Fatalf("非 ASCII 大小写不应归一: %q", got)
	}
}

// TestNormalizeSemanticsPreserved 语义保留属性测试（Story 4.3.1.1 T3；
// docs/design/04 §2 保留清单逐项：否定词/数字/单位/型号/药品符号/语义标点）。
func TestNormalizeSemanticsPreserved(t *testing.T) {
	cases := []struct {
		name     string
		question string
		mustKeep []string
	}{
		{"否定词", "这个药 不能 与酒精同服，禁止 空腹服用", []string{"不能", "禁止"}},
		{"无/非否定", "该服务 无需 备案，非营利 使用免费", []string{"无需", "非营利"}},
		{"剂量单位", "成人每日剂量 500mg，静脉注射 10mL", []string{"500mg", "10mL"}},
		{"型号符号", "A-100 型号与 B_200 不兼容（旧版）", []string{"A-100", "B_200"}},
		{"药品符号", "维生素C 与 阿莫西林 同服间隔 ≥2h", []string{"阿莫西林", "≥2h"}},
		{"语义标点", "支持配置文件、环境变量；不支持--flag", []string{"；", "--flag"}},
		{"数字精度", "超时设置为 1.5h 或 4800ms", []string{"1.5h", "4800ms"}},
		{"百分比与温度", "保存温度 2-8℃，湿度 ≤75%", []string{"2-8℃", "≤75%"}},
	}
	for _, tc := range cases {
		got := Normalize(tc.question, CaseASCII)
		for _, kw := range tc.mustKeep {
			kwNorm := Normalize(kw, CaseASCII)
			if !strings.Contains(got, kwNorm) {
				t.Errorf("%s: 规范化丢失关键词 %q → %q", tc.name, kw, got)
			}
		}
	}
}

// TestNormalizeIdempotent 规范化幂等：N(N(x)) == N(x)。
func TestNormalizeIdempotent(t *testing.T) {
	inputs := []string{" ＡＢ　C  xyz ", "服务  支持？", "Ａ-100  型号"}
	for _, in := range inputs {
		once := Normalize(in, CaseASCII)
		twice := Normalize(once, CaseASCII)
		if once != twice {
			t.Errorf("不幂等: %q → %q → %q", in, once, twice)
		}
	}
}

// TestQuestionDigest 稳定且不同问题不同摘要。
func TestQuestionDigest(t *testing.T) {
	a := QuestionDigest("服务支持哪些配置方式", CaseASCII)
	b := QuestionDigest("服务支持哪些配置方式", CaseASCII)
	if a != b || len(a) != 24 {
		t.Fatalf("摘要应稳定 24 hex: %q vs %q", a, b)
	}
	if QuestionDigest("另一个问题", CaseASCII) == a {
		t.Fatal("不同问题应不同摘要")
	}
	// 形态等价问题（首尾空白/全角空格/大小写）同摘要
	x := QuestionDigest(Normalize("　Service 支持哪些配置方式 ", CaseASCII), CaseASCII)
	y := QuestionDigest(Normalize("service 支持哪些配置方式", CaseASCII), CaseASCII)
	if x != y {
		t.Fatal("形态等价问题应同摘要")
	}
}
