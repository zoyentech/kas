package config

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestDurationParse 表驱动覆盖 docs/02 出现的全部 duration 形态与非法形态
// （Story 2.1.1.1 T3）。
func TestDurationParse(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{in: "4800ms", want: 4800 * time.Millisecond},
		{in: "400ms", want: 400 * time.Millisecond},
		{in: "1.5h", want: 90 * time.Minute},
		{in: "90s", want: 90 * time.Second},
		{in: "720h", want: 720 * time.Hour},
		{in: "120s", want: 2 * time.Minute},
		{in: "1ns", want: time.Nanosecond},
		{in: "0s", want: 0},
		{in: "0", want: 0, wantErr: true},           // 裸数字：必须是 duration 字符串
		{in: "-5s", want: -5 * time.Second},         // 负值语法合法，由校验层拒绝（必须为正）
		{in: "abc", wantErr: true},                  // 非法串
		{in: "4800 ms", wantErr: true},              // 带空格
		{in: "9999999999999999999h", wantErr: true}, // 溢出 time.Duration
	}
	for _, tc := range tests {
		var got struct {
			D Duration `yaml:"d"`
		}
		err := yaml.Unmarshal([]byte("d: "+tc.in), &got)
		if tc.wantErr {
			if err == nil {
				t.Errorf("d: %q 应被拒绝，实际得到 %s", tc.in, got.D)
			} else if !strings.Contains(err.Error(), tc.in) {
				t.Errorf("d: %q 错误应含原值，实际: %v", tc.in, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("d: %q 应合法，实际错误: %v", tc.in, err)
			continue
		}
		if time.Duration(got.D) != tc.want {
			t.Errorf("d: %q = %s, want %s", tc.in, got.D, tc.want)
		}
	}
}

// TestDurationNullYAML 显式 null 视为 0。
func TestDurationNullYAML(t *testing.T) {
	var got struct {
		D Duration `yaml:"d"`
	}
	if err := yaml.Unmarshal([]byte("d: null"), &got); err != nil {
		t.Fatalf("null duration 应为 0: %v", err)
	}
	if got.D != 0 {
		t.Fatalf("null duration 应为 0，实际 %s", got.D)
	}
}

// TestDurationRoundTrip YAML 形态字符串往返稳定（MarshalYAML）。
func TestDurationRoundTrip(t *testing.T) {
	b, err := yaml.Marshal(D(90 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "1h30m0s" {
		t.Fatalf("MarshalYAML = %q, want 1h30m0s", b)
	}
}
