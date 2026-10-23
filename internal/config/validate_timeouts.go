package config

import "time"

// 预算规则集（Story 2.1.2.1，F-001）。

// minReserveDefault 是 reserve 最小组装预算默认值（O-5，200ms；ADR-0002）。
const minReserveDefault = 200 * time.Millisecond

// timeoutFields 是五段字段名（错误路径与 Stages() 顺序一致）。
var timeoutFields = [...]string{
	"timeouts.preliminary", "timeouts.first_round", "timeouts.rejudge",
	"timeouts.supplement", "timeouts.reserve",
}

// ruleTimeoutsPositive 拒绝任何非正数预算段与 total（含 0）。
func ruleTimeoutsPositive(cfg Config) []Violation {
	var vs []Violation
	if cfg.Timeouts.Total <= 0 {
		vs = append(vs, newViolation("timeouts.total", "必须为正数，当前 %s", cfg.Timeouts.Total))
	}
	for i, s := range cfg.Timeouts.Stages() {
		if s <= 0 {
			vs = append(vs, newViolation(timeoutFields[i], "必须为正数，当前 %s", s))
		}
	}
	return vs
}

// ruleTimeoutsOverflow 拒绝五段加法溢出（int64 纳秒域；每段 ~2^62 级别的
// 构造在相加前后符号比较中暴露）。
func ruleTimeoutsOverflow(cfg Config) []Violation {
	if _, ok := cfg.Timeouts.Sum(); !ok {
		return []Violation{newViolation("timeouts", "五段之和溢出 int64 纳秒域（各段: %s/%s/%s/%s/%s）",
			cfg.Timeouts.Preliminary, cfg.Timeouts.FirstRound, cfg.Timeouts.Rejudge,
			cfg.Timeouts.Supplement, cfg.Timeouts.Reserve)}
	}
	return nil
}

// ruleTimeoutsEquation F-001 核心：total 与五段之和精确相等（纳秒比较，
// ±1ns 偏差即拒绝；错误给出两侧数值与差值）。
func ruleTimeoutsEquation(cfg Config) []Violation {
	sum, ok := cfg.Timeouts.Sum()
	if !ok {
		return nil // 已由 ruleTimeoutsOverflow 报告
	}
	if cfg.Timeouts.Total != sum {
		diff := cfg.Timeouts.Total - sum
		return []Violation{newViolation("timeouts.total",
			"total %s ≠ 五段之和 %s（差 %s）；预算等式 total = preliminary + first_round + rejudge + supplement + reserve 必须精确成立",
			cfg.Timeouts.Total, sum, diff)}
	}
	return nil
}

// ruleHostMaxTotal O-4：宿主 total 预算上限；未配置（nil）则不校验。
func ruleHostMaxTotal(cfg Config) []Violation {
	if cfg.Timeouts.HostMaxTotal != nil && cfg.Timeouts.Total > *cfg.Timeouts.HostMaxTotal {
		return []Violation{newViolation("timeouts.total",
			"total %s 超过宿主上限 host_max_total %s", cfg.Timeouts.Total, *cfg.Timeouts.HostMaxTotal)}
	}
	return nil
}

// ruleMinReserve O-5：reserve ≥ 最小组装预算（默认 200ms，min_reserve 可覆盖）。
func ruleMinReserve(cfg Config) []Violation {
	min := Duration(minReserveDefault)
	if cfg.Timeouts.MinReserve != nil {
		min = *cfg.Timeouts.MinReserve
	}
	if cfg.Timeouts.Reserve < min {
		return []Violation{newViolation("timeouts.reserve",
			"reserve %s 小于最小组装预算 %s（O-5 默认 200ms，可经 timeouts.min_reserve 配置覆盖，见 ADR-0002）",
			cfg.Timeouts.Reserve, min)}
	}
	return nil
}
