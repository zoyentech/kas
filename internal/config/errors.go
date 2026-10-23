package config

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidConfig 是全部配置解析/校验错误的哨兵（根包 kas.ErrInvalidConfig
// 即本值）。errors.Is(err, ErrInvalidConfig) 对所有配置错误成立。
var ErrInvalidConfig = errors.New("kas: invalid config")

// Violation 是单条配置违规：字段路径 + 原因。错误信息只含路径与原因，
// 不含 secret、问题正文或整份配置回显（Story 2.1.1.2 T3 规范）。
type Violation struct {
	Path   string // 如 "timeouts.total"
	Reason string // 如 "4800ms ≠ 五段之和 4801ms（差 1ms）"
}

// Error 使 Violation 可作为 error 使用。
func (v Violation) Error() string {
	return fmt.Sprintf("kas/config: %s: %s", v.Path, v.Reason)
}

// ValidationError 聚合一次校验的全部违规（逐条列出而非首条，Story 2.1.1.2 T2）。
type ValidationError struct {
	Violations []Violation
}

// Error 多行输出全部违规，每行一条。
func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "kas/config: 无违规"
	}
	lines := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		lines = append(lines, v.Error())
	}
	return strings.Join(lines, "\n")
}

// Unwrap 使 errors.Is(err, ErrInvalidConfig) 成立。
func (e *ValidationError) Unwrap() error { return ErrInvalidConfig }

// newViolation 构造单条违规；format 参数不得包含 secret/整份配置。
func newViolation(path, format string, args ...interface{}) Violation {
	return Violation{Path: path, Reason: fmt.Sprintf(format, args...)}
}

// singleError 将单条违规包装为 error。
func singleError(path, format string, args ...interface{}) error {
	return &ValidationError{Violations: []Violation{newViolation(path, format, args...)}}
}
