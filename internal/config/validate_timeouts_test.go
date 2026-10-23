package config

import (
	"math"
	"strings"
	"testing"
	"time"
)

// TestTimeoutsEquation F-001：±1ns 偏差即拒绝；错误展示两侧数值与差值
// （Story 2.1.2.1 T1）。
func TestTimeoutsEquation(t *testing.T) {
	tests := []struct {
		name    string
		total   time.Duration
		wantErr bool
	}{
		{"exact", 4800 * time.Millisecond, false},
		{"plus_1ns", 4800*time.Millisecond + 1, true},
		{"minus_1ns", 4800*time.Millisecond - 1, true},
		{"far_off", 1 * time.Second, true},
	}
	for _, tc := range tests {
		cfg := validConfig()
		cfg.Timeouts.Total = D(tc.total)
		err := Validate(cfg)
		if tc.wantErr && err == nil {
			t.Errorf("%s: 应拒绝", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("%s: 应通过，实际: %v", tc.name, err)
		}
	}
}

// TestTimeoutsEquationErrorShowsDiff 差值出现在错误中。
func TestTimeoutsEquationErrorShowsDiff(t *testing.T) {
	cfg := validConfig()
	cfg.Timeouts.Total = D(4800*time.Millisecond + 1)
	err := Validate(cfg)
	if err == nil {
		t.Fatal("应拒绝")
	}
	if got := err.Error(); !contains(got, "4800ms1ns") && !contains(got, "4.800000001s") {
		t.Fatalf("错误应含 total 与差值，实际: %s", got)
	}
}

// TestTimeoutsPositive 全部为 0 的配置被拒（Story 2.1.2.1 T2）。
func TestTimeoutsPositive(t *testing.T) {
	cfg := validConfig()
	cfg.Timeouts.Preliminary = 0
	cfg.Timeouts.Rejudge = D(-1)
	err := Validate(cfg)
	if err == nil {
		t.Fatal("非正数预算必须拒绝")
	}
	got := err.Error()
	if !contains(got, "timeouts.preliminary") || !contains(got, "timeouts.rejudge") {
		t.Fatalf("应定位到两个非法段，实际: %s", got)
	}
}

// TestTimeoutsOverflow 每段 ~2^62 的构造溢出被拒。
func TestTimeoutsOverflow(t *testing.T) {
	cfg := validConfig()
	big := D(time.Duration(math.MaxInt64 / 3))
	cfg.Timeouts = TimeoutBudget{
		Total: big, Preliminary: big, FirstRound: big,
		Rejudge: big, Supplement: big, Reserve: big,
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("溢出必须拒绝")
	} else if !contains(err.Error(), "溢出") {
		t.Fatalf("错误应说明溢出，实际: %s", err)
	}
}

// TestHostMaxTotal O-4：未配置不校验；配置后超限拒绝、等于边界通过。
func TestHostMaxTotal(t *testing.T) {
	cfg := validConfig()
	// 未配置：通过
	if err := Validate(cfg); err != nil {
		t.Fatalf("未配置 host_max_total 不应校验: %v", err)
	}
	// 配置为 4700ms：total 4800ms 超限
	h := D(4700 * time.Millisecond)
	cfg.Timeouts.HostMaxTotal = &h
	if err := Validate(cfg); err == nil || !contains(err.Error(), "host_max_total") {
		t.Fatalf("超宿主上限应拒绝，实际: %v", err)
	}
	// 边界：4800ms == total 通过
	h2 := D(4800 * time.Millisecond)
	cfg.Timeouts.HostMaxTotal = &h2
	if err := Validate(cfg); err != nil {
		t.Fatalf("等于上限应通过，实际: %v", err)
	}
}

// TestMinReserve O-5：reserve=100ms 默认被拒；min_reserve 覆盖后按新阈值判定。
func TestMinReserve(t *testing.T) {
	cfg := validConfig()
	// 构造 reserve=100ms 且 total=4300ms 的自洽配置（等式仍成立）
	cfg.Timeouts.Reserve = D(100 * time.Millisecond)
	cfg.Timeouts.Total = D(4300 * time.Millisecond)
	if err := Validate(cfg); err == nil || !contains(err.Error(), "最小组装预算") {
		t.Fatalf("reserve 低于默认下限应拒绝，实际: %v", err)
	}
	// 覆盖阈值 50ms：通过
	m := D(50 * time.Millisecond)
	cfg.Timeouts.MinReserve = &m
	if err := Validate(cfg); err != nil {
		t.Fatalf("min_reserve 覆盖后应通过，实际: %v", err)
	}
	// 默认 200ms 恰好等于下限：通过
	cfg2 := validConfig()
	cfg2.Timeouts.Reserve = D(200 * time.Millisecond)
	cfg2.Timeouts.Total = D(4400 * time.Millisecond)
	if err := Validate(cfg2); err != nil {
		t.Fatalf("reserve=200ms 等于默认下限应通过，实际: %v", err)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
