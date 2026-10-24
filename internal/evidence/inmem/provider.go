// Package inmem 提供可编程的 in-memory EvidenceProvider 参考实现。
// 仅测试/示例用途，非生产检索实现（Story 4.1.1.2 T1）。
package inmem

import (
	"context"
	"sort"
	"sync"

	"github.com/zoyentech/kas/internal/domain"
)

// Provider 是可编程返回指定 Evidence 集的参考实现：按 Query 过滤
// （namespace 由构造方在数据侧保证）、score 降序、截断 top_k；记录调用
// 与收到 ctx 供断言。
type Provider struct {
	mu       sync.Mutex
	items    []domain.Evidence
	emptyFor map[string]bool
	calls    int
	ctxs     []context.Context
}

// New 构建参考 provider。
func New(items []domain.Evidence) *Provider {
	return &Provider{items: append([]domain.Evidence(nil), items...), emptyFor: map[string]bool{}}
}

// SetEmptyFor 声明这些 namespace 返回空集（契约套件驱动空结果断言）。
func (p *Provider) SetEmptyFor(namespaces ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ns := range namespaces {
		p.emptyFor[ns] = true
	}
}

// Search 返回 score 降序前 top_k 条。
func (p *Provider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	p.ctxs = append(p.ctxs, ctx)
	if p.emptyFor[in.Namespace] {
		return domain.EvidenceResult{}, nil
	}
	out := append([]domain.Evidence(nil), p.items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if in.TopK > 0 && len(out) > in.TopK {
		out = out[:in.TopK]
	}
	return domain.EvidenceResult{Evidence: out}, nil
}

// Health 恒报 healthy（数据由构造方注入，无外部依赖）。
func (p *Provider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// Calls 返回调用计数。
func (p *Provider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// SetItems 替换返回集（测试中动态调整）。
func (p *Provider) SetItems(items []domain.Evidence) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append([]domain.Evidence(nil), items...)
}
