package contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/domain"
)

// EvidenceProviderSuite 是 EvidenceProvider 的合规契约套件（Story 4.1.1.1 T2）。
// 实现方在自己的 _test.go 一行接入：
//
//	func TestMyProviderContract(t *testing.T) {
//		contract.EvidenceProviderSuite(t, func(t *testing.T) domain.EvidenceProvider {
//			return myprovider.New(…)
//		})
//	}
//
// 覆盖：正常检索、空结果、error 透传、ctx 取消及时（provider 必须响应
// ctx.Done()——阻塞超阈值的实现（忽略 ctx）会被套件拒绝）、Health 结构。
func EvidenceProviderSuite(t *testing.T, mk func(t *testing.T) domain.EvidenceProvider) {
	t.Helper()

	q := domain.EvidenceQuery{Question: "测试问题", Namespace: "ns", TopK: 5}

	t.Run("normal_search", func(t *testing.T) {
		p := mk(t)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		res, err := p.Search(ctx, q)
		if err != nil {
			t.Fatalf("正常检索不应报错: %v", err)
		}
		if len(res.Evidence) == 0 {
			t.Fatal("参考实现应返回非空结果（空结果见 empty_result 用例）")
		}
	})

	t.Run("empty_result_ok", func(t *testing.T) {
		p := mk(t)
		if es, ok := p.(EmptyResultSetter); ok {
			es.SetEmptyFor("ns-empty") // 可空集驱动的实现参与空结果断言
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		res, err := p.Search(ctx, domain.EvidenceQuery{
			Question: "无结果问题", Namespace: "ns-empty", TopK: 5,
		})
		if err != nil {
			t.Fatalf("空结果应返回空集而非错误: %v", err)
		}
		if _, ok := p.(EmptyResultSetter); ok && len(res.Evidence) != 0 {
			t.Fatalf("期望空集，实际 %d 条", len(res.Evidence))
		}
	})

	t.Run("ctx_cancel_timely", func(t *testing.T) {
		p := mk(t)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, _ = p.Search(ctx, q) // 忽略返回值：只断言及时返回
		if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
			t.Fatalf("provider 必须在 ctx 取消后及时返回（耗时 %s）；忽略 ctx.Done() 的实现违反契约", elapsed)
		}
	})

	t.Run("ctx_canceled_returns_promptly", func(t *testing.T) {
		// 已取消 ctx：Search 必须及时返回（错误或快速完成皆可），
		// 不得假装成功后阻塞。
		p := mk(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = p.Search(ctx, q)
		}()
		select {
		case <-done:
		case <-time.After(250 * time.Millisecond):
			t.Fatal("已取消 ctx 下 Search 不得阻塞（provider 须响应 ctx.Done()）")
		}
	})

	t.Run("health_structure", func(t *testing.T) {
		p := mk(t)
		h := p.Health(context.Background())
		switch h.Status {
		case domain.ComponentHealthy, domain.ComponentDegraded, domain.ComponentNotReady, domain.ComponentUnknown:
		default:
			t.Fatalf("Health.Status 必须属于枚举，实际 %q", h.Status)
		}
	})
}

// ErrPassthroughIsPreserved 校验实现不吞错误：包装实现复用。
func ErrPassthroughIsPreserved(t *testing.T, mk func(t *testing.T) domain.EvidenceProvider, sentinel error) {
	t.Helper()
	p := mk(t)
	if _, err := p.Search(context.Background(), domain.EvidenceQuery{Question: "触发错误", Namespace: "err"}); !errors.Is(err, sentinel) {
		t.Fatalf("provider 错误应透传（errors.Is 判定），实际: %v", err)
	}
}

// PromptOnCancel 检测实现是否在 ctx 取消后及时返回（预算 = ctx 超时 +
// 容差）。套件内部与"违约实现演示"共用（忽略 ctx.Done() 的实现返回 false）。
func PromptOnCancel(p domain.EvidenceProvider, budget time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	start := time.Now()
	_, _ = p.Search(ctx, domain.EvidenceQuery{Question: "probe", Namespace: "ns"})
	return time.Since(start) <= 5*budget
}

// EmptyResultSetter 是契约套件的可选扩展：实现方声明指定 namespace 返回
// 空集，使套件能驱动"空结果 ≠ 错误"断言（inmem 等可控实现支持）。
type EmptyResultSetter interface {
	SetEmptyFor(namespaces ...string)
}
