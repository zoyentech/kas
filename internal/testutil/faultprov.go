package testutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/domain"
)

// FaultMode 是可独立开关的故障模式（Story 4.1.1.2 T2）。
type FaultMode struct {
	// Err 非 nil 时 Search 返回该错误。
	Err error
	// Delay > 0 时先 sleep（可被 ctx 取消打断）。
	Delay time.Duration
	// Block 为 true 时挂起直到 ctx.Done（与阻塞 harness 互补：本开关
	// 作用于包装的任意 inner provider）。
	Block bool
	// TruncateTo > 0 时把结果截断到前 N 条（部分结果故障）。
	TruncateTo int
	// FailAfter 前 N 次成功，之后返回 Err（若 Err 非 nil）。
	FailAfter int64
}

// FaultyProvider 包装任意 EvidenceProvider 注入故障；每种故障可独立开关，
// 供 S5 降级链与 S7 验收矩阵复用。
type FaultyProvider struct {
	Inner domain.EvidenceProvider
	Mode  FaultMode
	mu    sync.Mutex
	calls int64
}

// NewFaultyProvider 包装 inner。
func NewFaultyProvider(inner domain.EvidenceProvider, mode FaultMode) *FaultyProvider {
	return &FaultyProvider{Inner: inner, Mode: mode}
}

// Search 按故障模式执行。
func (p *FaultyProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	p.mu.Lock()
	p.calls++
	n := p.calls
	p.mu.Unlock()

	m := p.Mode
	if m.Delay > 0 {
		select {
		case <-time.After(m.Delay):
		case <-ctx.Done():
			return domain.EvidenceResult{}, ctx.Err()
		}
	}
	if m.Block {
		<-ctx.Done()
		return domain.EvidenceResult{}, ctx.Err()
	}
	if m.FailAfter > 0 && n > m.FailAfter {
		return domain.EvidenceResult{}, m.errOrDefault()
	}
	if m.Err != nil {
		return domain.EvidenceResult{}, m.Err
	}
	res, err := p.Inner.Search(ctx, in)
	if err != nil {
		return domain.EvidenceResult{}, err
	}
	if m.TruncateTo > 0 && len(res.Evidence) > m.TruncateTo {
		res.Evidence = res.Evidence[:m.TruncateTo]
	}
	return res, nil
}

// Health 委托 inner。
func (p *FaultyProvider) Health(ctx context.Context) domain.ComponentHealth {
	return p.Inner.Health(ctx)
}

// Calls 返回调用计数。
func (p *FaultyProvider) Calls() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

var errDefaultFault = errors.New("testutil: injected fault")

func (m FaultMode) errOrDefault() error {
	if m.Err != nil {
		return m.Err
	}
	return errDefaultFault
}
