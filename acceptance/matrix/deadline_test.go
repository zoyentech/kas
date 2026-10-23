package matrix

import (
	"context"
	"testing"
	"time"

	kas "github.com/kas-runtime/kas"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

// 矩阵行 15：provider 阻塞——ctx cancellation 传播；不启动越过 root
// deadline 的阶段。
func TestRowProviderBlocked(t *testing.T) {
	block := testutil.NewBlockEvidenceProvider()
	c := newClient(t, block, nil, nil)
	start := time.Now()
	resp, err := c.Answer(context.Background(), req("阻塞 provider 问题"))
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 600*time.Millisecond {
		t.Fatalf("请求不得越过 root deadline（400ms）: %s", elapsed)
	}
	if !resp.Degraded || resp.Action != kas.ActionAbstain {
		t.Fatalf("阻塞 provider 应安全降级: %+v", resp)
	}
	if block.Calls.Load() != 1 {
		t.Fatalf("阻塞调用计数 = %d", block.Calls.Load())
	}
	// 阻塞后阶段不再启动：决策 provider 未注入即链降级（调用计数=检索侧 1 次）
}

// root 越界守卫场景级：root 已过期的 caller ctx 下任何 provider 调用为 0。
func TestRowRootExpiredNoCalls(t *testing.T) {
	block := testutil.NewBlockEvidenceProvider()
	c := newClient(t, block, nil, nil)
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	resp, err := c.Answer(expired, req("root 过期问题"))
	if err != nil {
		t.Fatal(err)
	}
	if block.Calls.Load() != 0 {
		t.Fatalf("root 过期后 provider 调用应为 0: %d", block.Calls.Load())
	}
	if resp.Action != kas.ActionAbstain || !resp.Degraded {
		t.Fatalf("root 过期应安全降级 abstain: %+v", resp)
	}
}

// caller deadline 更早不可延长（F-002 场景级）。
func TestRowCallerDeadlineNotExtended(t *testing.T) {
	slow := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
	c := newClient(t, slow, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _ = c.Answer(ctx, req("caller 更早问题"))
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Fatalf("caller deadline 不可延长: %s", elapsed)
	}
}
