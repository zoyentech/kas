package kas

import "errors"

// 哨兵错误（docs/design/03 §6）：至少区分配置无效、请求无效、超时、
// Evidence/决策不可用、provider 契约违规与 SDK 已关闭。错误原文不直接
// 返回用户；错误信息含字段路径，不含 secret、问题正文与身份明文。
var (
	// ErrInvalidConfig 表示配置解析或启动校验失败（含 Reload 时的新配置）。
	ErrInvalidConfig = errors.New("kas: invalid config")
	// ErrInvalidRequest 表示请求参数不满足前置条件（如空问题、缺 principal）。
	ErrInvalidRequest = errors.New("kas: invalid request")
	// ErrDeadlineExceeded 表示请求在 root/阶段 deadline 内未完成。
	ErrDeadlineExceeded = errors.New("kas: deadline exceeded")
	// ErrEvidenceUnavailable 表示必需的 EvidenceProvider 不可用或全部失败。
	ErrEvidenceUnavailable = errors.New("kas: evidence unavailable")
	// ErrDecisionUnavailable 表示决策链（含降级链）不可用。
	ErrDecisionUnavailable = errors.New("kas: decision unavailable")
	// ErrProviderContract 表示 provider 返回违反契约的响应（未知 action、
	// 越界目标组、NaN/Inf confidence 等）。
	ErrProviderContract = errors.New("kas: provider contract violation")
	// ErrClosed 表示 Close 之后调用 Answer/Reload 等方法。
	ErrClosed = errors.New("kas: client closed")
)

// TODO(S3): 错误分类与 wrap 语义在 Story 3.1.1.2 细化（errors.Is/As 可判定）。
