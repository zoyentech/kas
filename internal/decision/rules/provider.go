// Package rules 实现确定性规则引擎（docs/design/04 §3）：条件完整、
// 输出有限 action、按声明序首个命中生效（语义见 Evaluate godoc）。
// 同一引擎双用途：降级链决策组件 + 编排器初判 hook。
package rules

import (
	"context"
	"fmt"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
)

// Context 是规则求值上下文：输入候选组 + 可选上游决策（链内上一级结果）。
type Context struct {
	Input domain.DecisionInput
	// Prior 上游 provider 的有效决策（nil = 无上游，如 Jev 失败后的规则级）。
	Prior *domain.Decision
}

// Condition 是条件接口：返回是否命中。实现必须纯函数（确定性）。
type Condition interface {
	Name() string
	Match(ctx Context) bool
}

// validEvidenceZero：全部候选组合计有效证据数为 0（F-010 无据禁答防线）。
type validEvidenceZero struct{}

func (validEvidenceZero) Name() string { return config.CondValidEvidenceZero }
func (validEvidenceZero) Match(c Context) bool {
	n := 0
	for _, g := range c.Input.Groups {
		n += len(g.Evidence)
	}
	return n == 0
}

// confidenceBelow：上游决策置信度低于阈值。
type confidenceBelow struct{ threshold float64 }

func (confidenceBelow) Name() string { return config.CondConfidenceBelowThreshold }
func (b confidenceBelow) Match(c Context) bool {
	return c.Prior != nil && c.Prior.Confidence < b.threshold
}

// builtinConditions 是内置条件注册表（配置校验的 KnownRuleConditions 镜像）。
func builtinConditions() map[string]func(cfg config.RuleConfig) (Condition, error) {
	return map[string]func(config.RuleConfig) (Condition, error){
		config.CondValidEvidenceZero: func(config.RuleConfig) (Condition, error) {
			return validEvidenceZero{}, nil
		},
		config.CondConfidenceBelowThreshold: func(cfg config.RuleConfig) (Condition, error) {
			if cfg.Threshold == nil {
				return nil, fmt.Errorf("条件 %s 缺少阈值", config.CondConfidenceBelowThreshold)
			}
			return confidenceBelow{threshold: *cfg.Threshold}, nil
		},
	}
}

// Engine 是规则引擎：按声明序求值，首个命中生效（Story 5.1.2.1 T2：
// 多规则命中的稳定裁决；同输入重复调用输出一致）。
type Engine struct {
	rules []compiledRule
}

type compiledRule struct {
	name      string
	action    domain.ActionType
	condition Condition
}

// New 由配置构建引擎；未知条件/非法 action 在构建期报错（启动校验）。
func New(cfg []config.RuleConfig) (*Engine, error) {
	conds := builtinConditions()
	e := &Engine{}
	for _, rc := range cfg {
		mk, ok := conds[rc.When]
		if !ok {
			return nil, fmt.Errorf("kas/rules: 未知条件 %q", rc.When)
		}
		cond, err := mk(rc)
		if err != nil {
			return nil, fmt.Errorf("kas/rules: 规则 %s: %w", rc.Name, err)
		}
		action := domain.ActionType(rc.Action)
		if !domain.ValidAction(action) {
			return nil, fmt.Errorf("kas/rules: 规则 %s: action %q 非法", rc.Name, rc.Action)
		}
		e.rules = append(e.rules, compiledRule{name: rc.Name, action: action, condition: cond})
	}
	return e, nil
}

// Evaluate 求值：按声明序遍历，首个命中返回 (action, 规则名, true)；
// 全不命中返回 (…, false)。确定性：同输入恒同输出。
func (e *Engine) Evaluate(ctx Context) (domain.ActionType, string, bool) {
	for _, r := range e.rules {
		if r.condition.Match(ctx) {
			return r.action, r.name, true
		}
	}
	return "", "", false
}

// Provider 把引擎适配为 DecisionProvider（降级链组件，Story 5.1.2.1 T3）。
type Provider struct {
	engine *Engine
}

// NewProvider 构建 rules 决策 provider。
func NewProvider(cfg []config.RuleConfig) (*Provider, error) {
	e, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return &Provider{engine: e}, nil
}

// Decide 求值规则；命中输出 Decision（Provider=rules，ReasonCode=规则名），
// 未命中返回 ErrNoMatch 由降级链继续（错误哨兵不外泄用户面）。
func (p *Provider) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	action, name, matched := p.engine.Evaluate(Context{Input: in})
	if !matched {
		return domain.Decision{}, ErrNoMatch
	}
	return domain.Decision{
		Action:     action,
		ReasonCode: name,
		Provider:   "rules",
		Degraded:   true,
	}, nil
}

// Health 规则引擎无外部依赖，恒 healthy。
func (p *Provider) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// ErrNoMatch 表示全部规则未命中（链继续下一级）。
var ErrNoMatch = fmt.Errorf("kas/rules: no rule matched")

// PreliminaryJudge 适配：规则引擎作初判 hook（docs/01 §4 第 4 步）。
// 命中 clarify/abstain 规则 → 对应初判输出；否则 continue。
type PreliminaryJudge struct{ engine *Engine }

// NewPreliminary 由配置构建初判。
func NewPreliminary(cfg []config.RuleConfig) (*PreliminaryJudge, error) {
	e, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return &PreliminaryJudge{engine: e}, nil
}

// Judge 初判：仅 evaluate 输入型条件（无 Prior）。
func (j *PreliminaryJudge) Judge(ctx context.Context, in domain.DecisionInput) (domain.ActionType, string, bool) {
	return j.engine.Evaluate(Context{Input: in})
}
