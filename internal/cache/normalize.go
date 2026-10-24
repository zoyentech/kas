package cache

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// CasePolicy 是大小写归一策略（Story 4.3.1.1 T2）：
//   - CaseOff：不做大小写归一
//   - CaseASCII（默认）：仅 ASCII 字母小写化；非 ASCII 大小写不受影响
//     （规避土耳其语 i 等区域陷阱；中文无大小写不受影响）
type CasePolicy int

// 大小写归一策略枚举（CaseASCII 为默认）。
const (
	CaseOff CasePolicy = iota
	CaseASCII
)

// Normalize 规范化问题文本（docs/design/04 §2）：
// Unicode NFKC → 首尾 trim → 连续空白折叠为单个空格 →（可选）ASCII 小写。
// 保留否定词、数字、单位、型号/药品符号与语义标点——本函数只做形态归一，
// 绝不删除或改写任何非空白字符（语义保留由 normalize_property_test.go 证明）。
func Normalize(q string, policy CasePolicy) string {
	s := norm.NFKC.String(q)
	s = strings.TrimSpace(s)
	s = collapseWhitespace(s)
	if policy == CaseASCII {
		s = lowerASCII(s)
	}
	return s
}

// collapseWhitespace 将连续空白（含全角空格经 NFKC 后的形态）折叠为单个空格。
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if isSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0:
		return true
	}
	return false
}

// lowerASCII 仅小写化 ASCII 字母。
func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// QuestionDigest 返回规范化问题的 SHA-256 前 24 hex 摘要。
// cache key 只存摘要，不保留问题原文（docs/design/04 §2）。
func QuestionDigest(normalized string, policy CasePolicy) string {
	return digestHex(Normalize(normalized, policy))[:24]
}

func digestHex(s string) string {
	return sha256Hex([]byte(s))
}
