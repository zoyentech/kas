package config

import (
	"errors"
	"strings"
	"testing"
)

// TestStrictDecodeRejectsUnknownField 未知字段拒绝且错误定位到字段名
// （Story 2.1.1.1 T2；docs/04 §1 拒绝项"未知配置字段"）。
func TestStrictDecodeRejectsUnknownField(t *testing.T) {
	bad := strings.Replace(minimalValidYAML, "top_k: 8", "topk: 8", 1)
	_, err := LoadConfig(strings.NewReader(bad))
	if err == nil {
		t.Fatal("未知字段 topk 必须被拒绝")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("错误应满足 errors.Is(ErrInvalidConfig)，实际: %v", err)
	}
	if !strings.Contains(err.Error(), "topk") {
		t.Fatalf("错误应定位到字段名 topk，实际:\n%v", err)
	}
}

// TestStrictDecodeAcceptsValid 合法配置零误报。
func TestStrictDecodeAcceptsValid(t *testing.T) {
	if _, err := LoadConfig(strings.NewReader(minimalValidYAML)); err != nil {
		t.Fatalf("合法配置不应误报:\n%v", err)
	}
}

// TestDecodeEmptyInput 空配置拒绝。
func TestDecodeEmptyInput(t *testing.T) {
	_, err := LoadConfig(strings.NewReader(""))
	if err == nil || !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("空配置应拒绝为 ErrInvalidConfig，实际: %v", err)
	}
}

// TestDecodeBrokenYAML 损坏 YAML 分类返回（含行号定位）。
func TestDecodeBrokenYAML(t *testing.T) {
	_, err := LoadConfig(strings.NewReader("timeouts: [unclosed"))
	if err == nil || !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("损坏 YAML 应分类为 ErrInvalidConfig，实际: %v", err)
	}
}

// TestDecodeNilReader nil reader 拒绝。
func TestDecodeNilReader(t *testing.T) {
	if _, err := LoadConfig(nil); err == nil {
		t.Fatal("nil reader 应拒绝")
	}
}

// TestDecodeMultiDocument 多文档 YAML 拒绝（只允许单文档）。
func TestDecodeMultiDocument(t *testing.T) {
	docs := minimalValidYAML + "\n---\n" + minimalValidYAML
	_, err := LoadConfig(strings.NewReader(docs))
	if err == nil {
		t.Fatal("多文档 YAML 应拒绝（第二个文档残余未读）")
	}
}
