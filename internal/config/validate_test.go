package config

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateAggregatesAllViolations 多条违规逐条列出而非首条
// （Story 2.1.1.2 T2）。
func TestValidateAggregatesAllViolations(t *testing.T) {
	cfg := validConfig()
	cfg.SchemaVersion = 2
	cfg.Timeouts.Total = D(100) // 不等式
	cfg.Evidence.TopK = 0
	err := Validate(cfg)
	if err == nil {
		t.Fatal("应有多条违规")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("应返回 *ValidationError，实际 %T", err)
	}
	paths := map[string]bool{}
	for _, v := range ve.Violations {
		paths[v.Path] = true
	}
	for _, want := range []string{"schema_version", "timeouts.total", "evidence.top_k"} {
		if !paths[want] {
			t.Errorf("违规聚合缺少 %s；实际全部: %v", want, ve.Violations)
		}
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Error("errors.Is(err, ErrInvalidConfig) 应成立")
	}
}

// TestValidateErrorFormatGolden 错误信息格式 golden：`kas/config: <path>: <reason>`。
func TestValidateErrorFormatGolden(t *testing.T) {
	cfg := validConfig()
	cfg.Timeouts.Total = D(1)
	err := Validate(cfg)
	if err == nil {
		t.Fatal("应有违规")
	}
	got := err.Error()
	if !strings.HasPrefix(got, "kas/config: timeouts.total: ") {
		t.Fatalf("错误格式应为 kas/config: <path>: <reason>，实际: %s", got)
	}
	if !strings.Contains(got, "1ns") || !strings.Contains(got, "4.8s") {
		t.Fatalf("预算等式错误应展示两侧数值（total 与 五段之和），实际: %s", got)
	}
}

// TestValidConfigPasses 基线配置零违规。
func TestValidConfigPasses(t *testing.T) {
	if err := Validate(validConfig()); err != nil {
		t.Fatalf("基线配置不应违规: %v", err)
	}
}

// validConfig 返回与 minimalValidYAML 等价的程序化配置。
func validConfig() Config {
	cfg, err := LoadConfig(strings.NewReader(minimalValidYAML))
	if err != nil {
		panic("minimalValidYAML 必须合法: " + err.Error())
	}
	return cfg
}
