package contract_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kas-runtime/kas/internal/contract"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/evidence/inmem"
)

// inmem 参考实现接入契约套件（Story 4.1.1.2 T1 验收：inmem 接入即跑通）。
func TestInmemProviderContract(t *testing.T) {
	contract.EvidenceProviderSuite(t, func(t *testing.T) domain.EvidenceProvider {
		return inmem.New([]domain.Evidence{
			{ID: "e1", SourceID: "s1", Score: 0.9},
			{ID: "e2", SourceID: "s1", Score: 0.5},
		})
	})
}

// errProvider 固定返回错误，验证错误透传契约。
var errInjected = errors.New("injected provider failure")

type errProvider struct{}

func (errProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	return domain.EvidenceResult{}, errInjected
}

func (errProvider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

func TestErrorPassthroughContract(t *testing.T) {
	contract.ErrPassthroughIsPreserved(t, func(t *testing.T) domain.EvidenceProvider {
		return errProvider{}
	}, errInjected)
}

// blockingProvider 挂起直到 ctx.Done（合规）。
type blockingProvider struct{}

func (blockingProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	<-ctx.Done()
	return domain.EvidenceResult{}, ctx.Err()
}

func (blockingProvider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// TestBlockingProviderPromptOnCancel 阻塞 harness 形态（挂起直到 ctx.Done）
// 满足"及时响应取消"契约谓词（完整套件适用于能返回结果的实现）。
func TestBlockingProviderPromptOnCancel(t *testing.T) {
	if !contract.PromptOnCancel(blockingProvider{}, 50*time.Millisecond) {
		t.Fatal("挂起直到 ctx.Done 的实现应被判定合规")
	}
}

// ignoreCtxProvider 违约实现：忽略 ctx 睡 2s —— 套件的 ctx_cancel_timely
// 必须抓住（Story 验收："故意违规实现（忽略 ctx）被套件抓住"）。
type ignoreCtxProvider struct{}

func (ignoreCtxProvider) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	time.Sleep(2 * time.Second) //nolint:staticcheck // 故意违约
	return domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "late"}}}, nil
}

func (ignoreCtxProvider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

func TestViolatingProviderCaught(t *testing.T) {
	// 违约实现（忽略 ctx 睡 2s）必须被 PromptOnCancel 抓住（Story 验收：
	// "故意违规实现（忽略 ctx）被套件抓住"；套件 ctx_cancel_timely 用例
	// 即用该谓词判定）。
	if contract.PromptOnCancel(ignoreCtxProvider{}, 50*time.Millisecond) {
		t.Fatal("忽略 ctx.Done() 的实现应被检测为违约")
	}
}
