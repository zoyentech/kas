package testutil

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/kas-runtime/kas/internal/domain"
)

// BlockEvidenceProvider 是"永不返回除非 ctx.Done"的假 EvidenceProvider
// （Story 3.2.1.2 T2 阻塞 provider 传播 harness）：记录被取消时刻与调用
// 计数，供契约套件与 S7 验收复用（docs/05 §3 口径：阶段截止后 ≤50ms 内
// provider 观察到取消）。
type BlockEvidenceProvider struct {
	Calls       atomic.Int32
	CancelledNS atomic.Int64 // unix-nano；0 表示未观察到取消
	CancelWait  chan struct{}
}

// NewBlockEvidenceProvider 构建阻塞 provider；cancelWait 在每次观察到
// ctx.Done 时关闭（可空）。
func NewBlockEvidenceProvider() *BlockEvidenceProvider {
	return &BlockEvidenceProvider{CancelWait: make(chan struct{})}
}

// Search 阻塞直至 ctx 取消（合规响应 ctx.Done()）。
func (p *BlockEvidenceProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	p.Calls.Add(1)
	<-ctx.Done()
	p.CancelledNS.Store(time.Now().UnixNano())
	close(p.CancelWait)
	return domain.EvidenceResult{}, ctx.Err()
}

// Health 恒报 healthy（隔离阻塞行为于 Search）。
func (p *BlockEvidenceProvider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// CancelledAt 返回观察到的取消时刻（未观察到返回零值）。
func (p *BlockEvidenceProvider) CancelledAt() time.Time {
	if ns := p.CancelledNS.Load(); ns != 0 {
		return time.Unix(0, ns)
	}
	return time.Time{}
}

// BlockDecisionProvider 是决策侧的阻塞假 provider。
type BlockDecisionProvider struct {
	Calls       atomic.Int32
	CancelledNS atomic.Int64
}

// Decide 阻塞直至 ctx 取消。
func (p *BlockDecisionProvider) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	p.Calls.Add(1)
	<-ctx.Done()
	p.CancelledNS.Store(time.Now().UnixNano())
	return domain.Decision{}, ctx.Err()
}

// Health 恒报 healthy。
func (p *BlockDecisionProvider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}
