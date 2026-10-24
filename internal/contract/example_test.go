package contract_test

import (
	"context"
	"testing"

	"github.com/zoyentech/kas/internal/contract"
)

// 本文件演示契约套件的"一行接入"模式（Story 1.2.2.2 T3）：
//
//	1. 契约包为每个领域接口提供 Suite 函数：func SomeInterfaceSuite(t *testing.T, mk func() SomeInterface)
//	2. 实现方在自己的 _test.go 中实现工厂（每个用例获得全新实例），调用套件：
//
//		func TestMyImplContract(t *testing.T) {
//			contract.SomeInterfaceSuite(t, func() SomeInterface { return NewMyImpl(…) })
//		}
//
//	套件统一覆盖：ctx 取消传播、超时语义、并发安全、契约错误分类。
//	Stage 4/5 为 EvidenceProvider / DecisionProvider / EvidenceCache 落地正式套件
//	（internal/contract/evidence.go、decision.go、cache.go）。
//
// 下面用一个最小接口演示该模式的形态（玩具接口仅用于演示，不进入生产代码）。

// Adder 是演示用最小接口。
type Adder interface {
	Add(ctx context.Context, a, b int) (int, error)
}

// adderImpl 是演示实现：ctx 取消时返回 ctx.Err()，模拟 provider 必须响应
// ctx.Done() 的契约。
type adderImpl struct{}

func (adderImpl) Add(ctx context.Context, a, b int) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		return a + b, nil
	}
}

// TestAdderContractSuite 演示实现方接入套件的方式：工厂每个用例新建实例。
func TestAdderContractSuite(t *testing.T) {
	contract.RunDemoSuite(t, func(t *testing.T) Adder {
		t.Helper()
		return adderImpl{}
	})
}
