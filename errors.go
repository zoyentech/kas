package kas

import "github.com/zoyentech/kas/internal/domain"

// 哨兵错误（docs/design/03 §6；唯一权威定义在 internal/domain）：
// errors.Is 可判定的七类。错误原文不直接返回用户——编排器输出固定枚举
// 文案；错误信息含字段路径，不含 secret、问题正文与身份明文。
var (
	// ErrInvalidConfig 表示配置解析或启动校验失败（含 Reload 时的新配置）。
	ErrInvalidConfig = domain.ErrInvalidConfig
	// ErrInvalidRequest 表示请求参数不满足前置条件（如空问题、缺 principal）。
	ErrInvalidRequest = domain.ErrInvalidRequest
	// ErrDeadlineExceeded 表示请求在 root/阶段 deadline 内未完成。
	ErrDeadlineExceeded = domain.ErrDeadlineExceeded
	// ErrEvidenceUnavailable 表示必需的 EvidenceProvider 不可用或全部失败。
	ErrEvidenceUnavailable = domain.ErrEvidenceUnavailable
	// ErrDecisionUnavailable 表示决策链（含降级链）不可用。
	ErrDecisionUnavailable = domain.ErrDecisionUnavailable
	// ErrProviderContract 表示 provider 返回违反契约的响应（未知 action、
	// 越界目标组、NaN/Inf confidence 等）。
	ErrProviderContract = domain.ErrProviderContract
	// ErrClosed 表示 Close 之后调用 Answer/Reload 等方法。
	ErrClosed = domain.ErrClosed
)
