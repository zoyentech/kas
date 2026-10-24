package health

import (
	"context"
	"sync"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
)

// Deps 是可探测组件集合（nil 组件跳过）。
type Deps struct {
	Evidence domain.EvidenceProvider
	Decision domain.DecisionProvider
	Extra    domain.HealthChecks // 宿主注入检查点
}

// Probe 并发探测全部组件（Story 6.1.1.1 T2）：每组件独立 check_timeout
// 派生 ctx；超时/失败组件标记 not_ready；慢组件不拖垮整体（总耗时 ≤
// check_timeout + ε）。探测不触发用户查询（各组件 Health 契约）。
// 结果同时以 HealthEvent 发布到 bus（可空）。
func Probe(parent context.Context, cfg config.HealthConfig, deps Deps, bus *obs.Bus) domain.HealthReport {
	var mu sync.Mutex
	components := map[string]domain.ComponentHealth{}

	probeOne := func(name string, fn func(ctx context.Context) domain.ComponentHealth) {
		ctx, cancel := context.WithTimeout(parent, cfg.CheckTimeout.Std())
		defer cancel()
		h := fn(ctx)
		if ctxErr := ctx.Err(); ctxErr != nil {
			h = domain.ComponentHealth{Status: domain.ComponentNotReady, Detail: "health check timeout"}
		}
		mu.Lock()
		components[name] = h
		mu.Unlock()
		if bus != nil {
			bus.Emit(obs.HealthEvent{Component: name, Status: string(h.Status)})
		}
	}

	var wg sync.WaitGroup
	probes := map[string]func(context.Context) domain.ComponentHealth{}
	if deps.Evidence != nil {
		probes["evidence_provider"] = deps.Evidence.Health
	}
	if deps.Decision != nil {
		probes["jev"] = deps.Decision.Health
	}
	for name, fn := range deps.Extra {
		probes[name] = fn
	}
	for name, fn := range probes {
		wg.Add(1)
		go func(name string, fn func(context.Context) domain.ComponentHealth) {
			defer wg.Done()
			probeOne(name, fn)
		}(name, fn)
	}
	wg.Wait()

	return Aggregate(Input{
		Components:                      components,
		Required:                        cfg.RequiredComponents,
		Optional:                        cfg.OptionalComponents,
		DegradedWhenOptionalUnavailable: cfg.DegradedWhenOptionalProviderUnavailable,
	})
}

// timeNow 便于测试替换。
var timeNow = time.Now
