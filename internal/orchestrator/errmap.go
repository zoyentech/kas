package orchestrator

import (
	"context"
	"errors"

	"github.com/kas-runtime/kas/internal/domain"
)

// MapToPublicError 将内部错误映射到公共错误面（Story 3.1.1.2 T2：
// 错误原文不直接返回用户）。公共错误串为固定枚举文案；上游错误体、
// prompt、思维链仅进入 trace/安全日志（ProviderErrorEvent/TraceEvent），
// 零外泄到 Response 或返回给宿主的 error。
func MapToPublicError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrInvalidRequest):
		return domain.ErrInvalidRequest
	case errors.Is(err, domain.ErrInvalidConfig):
		return domain.ErrInvalidConfig
	case errors.Is(err, domain.ErrDeadlineExceeded):
		return domain.ErrDeadlineExceeded
	case errors.Is(err, domain.ErrEvidenceUnavailable):
		return domain.ErrEvidenceUnavailable
	case errors.Is(err, domain.ErrDecisionUnavailable):
		return domain.ErrDecisionUnavailable
	case errors.Is(err, domain.ErrProviderContract):
		return domain.ErrProviderContract
	case errors.Is(err, domain.ErrClosed):
		return domain.ErrClosed
	case errors.Is(err, context.Canceled):
		return domain.ErrDeadlineExceeded
	case errors.Is(err, context.DeadlineExceeded):
		return domain.ErrDeadlineExceeded
	default:
		// 未知内部错误：归类为 evidence unavailable 不可行——保守映射为
		// 契约外的通用错误文案，不携带任何内部细节。
		return errInternal
	}
}

// errInternal 是未知内部错误的固定公共文案（不含内部细节）。
var errInternal = errors.New("kas: internal error")

// ReasonForError 返回受控降级 reason code（进入 Response.Degradation 与
// fallback 指标 label）。
func ReasonForError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, domain.ErrDeadlineExceeded):
		return domain.ReasonDeadlineExceeded
	case errors.Is(err, domain.ErrProviderContract):
		return domain.ReasonJevContractError
	case errors.Is(err, domain.ErrEvidenceUnavailable):
		return domain.ReasonEvidenceUnavailable
	case errors.Is(err, domain.ErrDecisionUnavailable):
		return domain.ReasonDecisionExhausted
	default:
		return domain.ReasonEvidenceUnavailable
	}
}
