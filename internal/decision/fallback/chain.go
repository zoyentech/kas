// Package fallback 实现降级链执行器（F-008；docs/design/04 §3）：
// 顺序固定 Jev 一次尝试 → 确定性规则 → 受约束决策缓存 → abstain/clarify。
// 每级错误分类捕获不中断链；degraded/reason 贯通；每级尝试发
// FallbackEvent（route/reason）；剩余预算不足即进入安全终态。
package fallback

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zoyentech/kas/internal/decision"
	"github.com/zoyentech/kas/internal/decision/rules"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
)

// Route 名（kas_decision_fallback_total route label 有界集合）。
const (
	RouteJev     = "jev"
	RouteRules   = "deterministic_rules"
	RouteCache   = "decision_cache"
	RouteAbstain = "abstain"
)

// Stage 是链的一级：provider + 路由名。
type Stage struct {
	Route    string
	Provider domain.DecisionProvider
}

// Chain 是降级链执行器（实现 domain.DecisionProvider）。
type Chain struct {
	Stages []Stage // 顺序即尝试顺序（docs/04 §3 固定：jev→rules→cache）
	Bus    *obs.Bus
	Clock  func() time.Time
	// guardrail 是确定性规则护栏（rules 引擎；nil = 无护栏）。
	guard *rules.Engine
	// CacheWriter 是决策缓存回填口（dcache.Provider；nil = 不回填）。
	// 仅回填首选级（Jev）成功且未被护栏覆盖的正向决策（禁写终态/低置信
	// 由 dcache.Put 内部规则保证）。
	CacheWriter interface {
		Put(ctx context.Context, in domain.DecisionInput, d domain.Decision)
	}
	// MinAttemptCost 进入下一级前的最小剩余预算（低于即终止到安全终态）。
	MinAttemptCost time.Duration
}

// New 构建链。stages 按固定顺序整理（order 乱序/缺级由注册集纠正：
// 只按 route 归位存在的级，注释化 docs/04 §3"顺序固定"）。
func New(stages []Stage, bus *obs.Bus, guard *rules.Engine) *Chain {
	byRoute := map[string]domain.DecisionProvider{}
	for _, s := range stages {
		if s.Provider != nil {
			byRoute[s.Route] = s.Provider
		}
	}
	ordered := make([]Stage, 0, len(byRoute))
	for _, route := range []string{RouteJev, RouteRules, RouteCache} {
		if p, ok := byRoute[route]; ok {
			ordered = append(ordered, Stage{Route: route, Provider: p})
		}
	}
	return &Chain{Stages: ordered, Bus: bus, Clock: time.Now, guard: guard, MinAttemptCost: time.Millisecond}
}

// Decide 按序执行降级链（Story 5.3.1.1）：
//  1. 每级调用 + 统一校验（decision.ValidateDecision）；失败分类记录
//     reason 续链，不中断
//  2. 上游成功后仍过确定性规则护栏：命中（无据禁答/低置信澄清）→ 规则
//     动作生效，degraded + rules_applied（可审计护栏，docs/01 §6）
//  3. 剩余预算不足 → 立即安全终态（不启动必超时的下一级）
//  4. 全部耗尽 → abstain + chain_exhausted
func (c *Chain) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	meta := obs.Meta{TraceID: in.TraceID, ConfigVersion: in.ConfigVersion, At: c.Clock()}
	var reasons []string

	for _, s := range c.Stages {
		// 预算守卫（Story 5.3.1.1 T3）
		if c.MinAttemptCost > 0 {
			if dl, ok := ctx.Deadline(); ok && time.Until(dl) < c.MinAttemptCost {
				c.emitFallback(meta, RouteAbstain, domain.ReasonDeadlineExceeded)
				return terminal(domain.ReasonDeadlineExceeded, append(reasons, domain.ReasonDeadlineExceeded)), nil
			}
		}
		if err := ctx.Err(); err != nil {
			c.emitFallback(meta, RouteAbstain, domain.ReasonDeadlineExceeded)
			return terminal(domain.ReasonDeadlineExceeded, append(reasons, domain.ReasonDeadlineExceeded)), nil
		}

		d, err := c.safeDecide(ctx, s, in)
		if err != nil {
			reason := reasonForStage(err, s.Route)
			reasons = appendReason(reasons, reason)
			c.emitFallback(meta, s.Route, reason)
			continue
		}
		if err := decision.ValidateDecision(in, d); err != nil {
			reasons = appendReason(reasons, domain.ReasonJevContractError)
			c.emitFallback(meta, s.Route, domain.ReasonJevContractError)
			continue
		}

		// 确定性规则护栏：仅对非规则级的上游结果（Jev）应用——规则级自身
		// 输出即规则结果，无需二次覆盖
		if s.Route == RouteJev && c.guard != nil {
			if action, ruleName, matched := c.guard.Evaluate(rules.Context{Input: in, Prior: &d}); matched {
				g := domain.Decision{
					Action:       action,
					ReasonCode:   ruleName,
					Provider:     "rules",
					Degraded:     true,
					Model:        d.Model,
					ModelVersion: d.ModelVersion,
					Degradation:  appendReasons(reasons, domain.ReasonRulesApplied),
				}
				c.emitFallback(meta, RouteRules, domain.ReasonRulesApplied)
				return g, nil
			}
		}
		if s.Route == RouteJev && c.CacheWriter != nil {
			c.CacheWriter.Put(ctx, in, d)
		}
		if s.Route != RouteJev {
			d.Degraded = true
			d.Degradation = appendReasons(reasons, domain.ReasonDecisionCacheHit)
		}
		return d, nil
	}

	// 链耗尽 → 安全终态
	c.emitFallback(meta, RouteAbstain, domain.ReasonDecisionExhausted)
	return terminal(domain.ReasonDecisionExhausted, appendReason(reasons, domain.ReasonDecisionExhausted)), nil
}

// Health 链健康 = 首选级（Jev）健康；链为空报 not_ready。
func (c *Chain) Health(ctx context.Context) domain.ComponentHealth {
	if len(c.Stages) == 0 {
		return domain.ComponentHealth{Status: domain.ComponentNotReady, Detail: "降级链为空"}
	}
	return c.Stages[0].Provider.Health(ctx)
}

// safeDecide 执行单级 provider 并隔离 panic（Story 8.5.1.1）：panic 视为该级
// 失败（契约违规大类），链继续下一级而非击穿进程。
func (c *Chain) safeDecide(ctx context.Context, s Stage, in domain.DecisionInput) (d domain.Decision, err error) {
	defer func() {
		if r := recover(); r != nil {
			d, err = domain.Decision{}, fmt.Errorf("%w: route=%s provider panicked (isolated)",
				domain.ErrProviderContract, s.Route)
		}
	}()
	return s.Provider.Decide(ctx, in)
}

func (c *Chain) emitFallback(meta obs.Meta, route, reason string) {
	if c.Bus != nil {
		c.Bus.Emit(obs.FallbackEvent{Meta: meta, Route: route, Reason: reason})
	}
}

func reasonForStage(err error, route string) string {
	switch {
	case errors.Is(err, domain.ErrDeadlineExceeded):
		return domain.ReasonDeadlineExceeded
	case errors.Is(err, domain.ErrProviderContract):
		return domain.ReasonJevContractError
	case errors.Is(err, rules.ErrNoMatch):
		return domain.ReasonRulesApplied // 规则未命中不算失败：链按序继续，reason 记录求值发生
	default:
		return domain.ReasonJevUnavailable
	}
}

func appendReason(reasons []string, reason string) []string {
	for _, r := range reasons {
		if r == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func terminal(reason string, reasons []string) domain.Decision {
	return domain.Decision{
		Action:      domain.ActionAbstain,
		ReasonCode:  reason,
		Provider:    "fallback",
		Degraded:    true,
		Degradation: appendReasons(reasons, reason),
	}
}

// appendReasons 追加并去重。
func appendReasons(reasons []string, reason string) []string {
	for _, r := range reasons {
		if r == reason {
			return reasons
		}
	}
	out := append([]string(nil), reasons...)
	return append(out, reason)
}
