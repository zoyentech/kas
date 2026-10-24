package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zoyentech/kas/internal/deadline"
	"github.com/zoyentech/kas/internal/decision"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/runtime"
)

// Answer 执行一次问答编排（Story 3.3.1.1 T1）：
// 校验 Request → 读 snapshot（一次并全程持有）→ 固定 config_version →
// 五阶段流水线 → 组装 Response。并发安全（无共享可变状态）。
func (e *Engine) Answer(ctx context.Context, req domain.Request) (domain.Response, error) {
	if err := domain.ValidateRequest(req); err != nil {
		return domain.Response{}, err
	}
	snap := e.Manager.Current() // 请求开始读取一次并全程持有（F-011）
	if snap == nil {
		return domain.Response{}, fmt.Errorf("kas: %w: 运行时快照未加载", domain.ErrInvalidConfig)
	}

	traceID := e.TraceID()
	now := e.Clock.Now()
	meta := obs.Meta{TraceID: traceID, ConfigVersion: snap.Version(), At: now}

	root, cancelRoot := deadline.NewRoot(ctx, now, snap.Budget().Total.Std())
	defer cancelRoot() // 全路径 cancel 保证

	st := &state{
		req:   req,
		snap:  snap,
		trace: traceID,
		meta:  meta,
	}

	start := e.Clock.Now()
	resp, err := e.runPipeline(root, st)
	e.Bus.Emit(obs.RequestEndEvent{
		Meta:     withAt(meta, e.Clock.Now()),
		Outcome:  outcomeOf(resp, err),
		Action:   string(resp.Action),
		Duration: e.Clock.Now().Sub(start),
		Degraded: resp.Degraded,
	})
	return resp, err
}

// state 是一次请求的阶段间中间状态（单请求私有，无并发访问）。
type state struct {
	req        domain.Request
	snap       *runtime.Snapshot
	trace      string
	meta       obs.Meta
	prelim     PreliminaryOutcome
	evidence   domain.EvidenceResult
	evErr      error
	decision   domain.Decision
	decErr     error
	supplement domain.EvidenceResult // S4：补检结果重新过滤钩子（docs/01 §4 第 8 步）
	degraded   bool
	reasons    []string
}

// 阶段名常量（指标 stage label 有界集合）。
const (
	StagePreliminary = "preliminary"
	StageFirstRound  = "first_round"
	StageRejudge     = "rejudge"
	StageSupplement  = "supplement"
	StageReserve     = "reserve"
)

// runPipeline 五阶段串联执行器（Story 3.3.1.1 T3）：每阶段只从 root 派生
// ctx（docs/04 §1），阶段结束立即 cancel 再派生下一阶段；每阶段启动前做
// root 越界守卫（Story 3.2.1.2 T3：不启动任何会越过 root deadline 的阶段）。
func (e *Engine) runPipeline(root *deadline.Root, st *state) (domain.Response, error) {
	budget := st.snap.Budget()

	// —— 初判 ——
	_, prelimErr := e.runStage(root, st, StagePreliminary, budget.Preliminary.Std(), "internal",
		func(ctx context.Context) error {
			p, err := e.Preliminary.Judge(ctx, PreliminaryInput{Request: st.req, ConfigVersion: st.snap.Version()})
			st.prelim = p
			return err
		})
	if prelimErr != nil {
		return e.assembleDegraded(root, st, domain.ReasonDeadlineExceeded)
	}
	switch outcome2 := st.prelim; outcome2.Kind {
	case PreliminaryClarify:
		st.decision = domain.Decision{Action: domain.ActionClarify, ReasonCode: outcome2.Reason}
		return e.assembleReserve(root, st)
	case PreliminaryAbstain:
		st.decision = domain.Decision{Action: domain.ActionAbstain, ReasonCode: outcome2.Reason}
		return e.assembleReserve(root, st)
	}

	// —— 首轮（cache/检索 → 候选组）——
	if _, err := e.runStage(root, st, StageFirstRound, budget.FirstRound.Std(), "evidence", func(ctx context.Context) error {
		return e.searchEvidence(ctx, st, false)
	}); err != nil {
		st.evErr = err
		st.degraded = true
		if isDeadlineErr(err) {
			st.reasons = append(st.reasons, domain.ReasonDeadlineExceeded)
		} else {
			st.reasons = append(st.reasons, domain.ReasonEvidenceUnavailable)
		}
	}

	// —— 复判（决策）——
	if _, err := e.runStage(root, st, StageRejudge, budget.Rejudge.Std(), "decision", func(ctx context.Context) error {
		return e.decide(ctx, st)
	}); err != nil {
		st.decErr = err
		st.degraded = true
		if isDeadlineErr(err) {
			st.reasons = append(st.reasons, domain.ReasonDeadlineExceeded)
		} else {
			st.reasons = append(st.reasons, domain.ReasonDecisionExhausted)
		}
	}

	// —— 补检（仅当决策要求且 root 未到期；不运行时不做预算重分配）——
	if st.decision.NeedSupplement && !root.Expired(e.Clock.Now()) {
		if _, err := e.runStage(root, st, StageSupplement, budget.Supplement.Std(), "evidence", func(ctx context.Context) error {
			// TODO(S4): 补检结果重新经过权限与有效期过滤（docs/01 §4 第 8 步）
			return e.searchEvidence(ctx, st, true)
		}); err != nil {
			st.reasons = append(st.reasons, domain.ReasonEvidenceUnavailable)
			st.degraded = true
		}
	}

	// —— 余量（引用检查、组装、编码）——
	return e.assembleReserve(root, st)
}

// runStage 执行单个阶段：root 守卫 → 从 root 派生阶段 ctx → 运行 →
// 立即 cancel → 发 StageEndEvent。deadline 超时发 StageDeadlineEvent 并返回
// ErrDeadlineExceeded。
func (e *Engine) runStage(root *deadline.Root, st *state, name string, budget time.Duration,
	provider string, fn func(ctx context.Context) error,
) (StageOutcome, error) {
	now := e.Clock.Now()
	// root 越界守卫（Story 3.2.1.2 T3）：root 已到期则不启动本阶段，
	// 任何 provider 调用计数为 0。
	if root.Expired(now) {
		e.Bus.Emit(obs.StageDeadlineEvent{Meta: withAt(st.meta, now), Stage: name, DeadlineStage: name})
		e.Bus.Emit(obs.StageEndEvent{
			Meta: withAt(st.meta, e.Clock.Now()), Stage: name, Provider: provider,
			Outcome: "deadline_exceeded", Duration: 0, Remaining: 0,
		})
		return StageOutcome{}, deadline.ExpiredErr(name)
	}
	stageStart := now
	stageCtx, cancelStage := root.Stage(name, stageStart, budget)
	defer cancelStage() // 阶段结束立即取消（全路径 cancel 保证）

	err := fn(stageCtx)
	cancelStage()

	elapsed := e.Clock.Now().Sub(stageStart)
	remaining := time.Duration(0)
	if d, ok := stageCtx.Deadline(); ok {
		remaining = d.Sub(e.Clock.Now())
	}
	outcome := StageOutcome{}
	if err != nil {
		if isDeadlineErr(err) {
			e.Bus.Emit(obs.StageDeadlineEvent{Meta: withAt(st.meta, e.Clock.Now()), Stage: name, DeadlineStage: name})
			e.Bus.Emit(obs.StageEndEvent{
				Meta: withAt(st.meta, e.Clock.Now()), Stage: name, Provider: provider,
				Outcome: "deadline_exceeded", Duration: elapsed, Remaining: remaining,
			})
			return outcome, err
		}
		e.Bus.Emit(obs.StageEndEvent{
			Meta: withAt(st.meta, e.Clock.Now()), Stage: name, Provider: provider,
			Outcome: "provider_error", Duration: elapsed, Remaining: remaining,
		})
		return outcome, err
	}
	e.Bus.Emit(obs.StageEndEvent{
		Meta: withAt(st.meta, e.Clock.Now()), Stage: name, Provider: provider,
		Outcome: "ok", Duration: elapsed, Remaining: remaining,
	})
	return outcome, nil
}

// StageOutcome 是阶段执行结果占位（S4/S5 扩展具体产物）。
type StageOutcome struct{}

// searchEvidence 执行 Evidence 检索（首轮与补检共用）：统一经 CachedProvider
// 缓存读写路径（docs/design/01 §4 第 5 步；第 8 步补检结果同样经该路径
// 重新权限+有效期过滤，过滤路径唯一不绕过）。deadline.Call 包装保证
// provider 违约也不越过阶段预算。
func (e *Engine) searchEvidence(ctx context.Context, st *state, supplement bool) error {
	if e.Evidence == nil {
		return errors.New("kas: evidence provider 未注入")
	}
	stage := StageFirstRound
	if supplement {
		stage = StageSupplement
	}
	snapCfg := st.snap.Config()
	provider := evidence.NewCachedProvider(e.Evidence, e.Cache, snapCfg.Evidence, e.Perm, e.Bus)
	provider.Clock = e.Clock.Now
	query := domain.EvidenceQuery{
		Question:         st.req.Question,
		Namespace:        st.req.Knowledge.Namespace,
		CorpusGeneration: st.req.Knowledge.CorpusGeneration,
		TopK:             snapCfg.Evidence.TopK,
		ValidAt:          e.Clock.Now(),
		Scope:            st.req.Principal,
		ConfigVersion:    st.snap.Version(),
		TraceID:          st.trace,
	}
	res, err := deadline.Call(stage, ctx, e.Clock.Now(), func(ctx context.Context) (domain.EvidenceResult, error) {
		return provider.Search(ctx, query)
	})
	if err != nil {
		e.emitProviderError(stage, "search", err, st)
		return err
	}
	if supplement {
		st.supplement = res
	} else {
		st.evidence = res
	}
	return nil
}

// decide 执行决策（复判；S5 在此挂降级链）。
func (e *Engine) decide(ctx context.Context, st *state) error {
	if e.Decision == nil {
		// 决策 provider 可选：S5 降级链接管（规则→缓存→abstain）。
		st.degraded = true
		st.reasons = append(st.reasons, domain.ReasonDecisionExhausted)
		return nil
	}
	dec, err := deadline.Call(StageRejudge, ctx, e.Clock.Now(), func(ctx context.Context) (domain.Decision, error) {
		return e.Decision.Decide(ctx, domain.DecisionInput{
			Question:           st.req.Question,
			Groups:             candidateGroups(st),
			ValidAt:            e.Clock.Now(),
			ConfigVersion:      st.snap.Version(),
			TraceID:            st.trace,
			ScopeFingerprint:   st.req.Principal.ScopeFingerprint,
			KnowledgeCorpusGen: st.req.Knowledge.CorpusGeneration,
		})
	})
	if err != nil {
		e.emitProviderError(StageRejudge, "decide", err, st)
		return err
	}
	st.decision = dec
	if dec.Degraded {
		st.degraded = true
		for _, r := range dec.Degradation {
			st.reasons = appendUnique(st.reasons, r)
		}
	}
	// F-009：决策版本链进 trace（provider/model/model_version/候选数/action）
	e.Bus.Emit(obs.DecisionTraceEvent{
		Meta: withAt(st.meta, e.Clock.Now()), Provider: dec.Provider,
		Model: dec.Model, ModelVersion: dec.ModelVersion,
		FallbackRoute: dec.Provider, CandidateCount: len(st.evidence.Evidence),
		Action: string(dec.Action), ReasonCode: dec.ReasonCode, Confidence: dec.Confidence,
	})
	return nil
}

// candidateGroups 由（合并后的最终）Evidence 构造候选组（v1.2 简化为单组；
// S4 属性白名单经 decision.FilterAttributes 应用）。
func candidateGroups(st *state) []domain.CandidateGroup {
	ev := st.evidence.Evidence
	ev = append(append([]domain.Evidence(nil), ev...), st.supplement.Evidence...)
	if len(ev) == 0 {
		return nil
	}
	g := domain.CandidateGroup{ID: "g1", Score: ev[0].Score}
	for _, item := range ev {
		g.Evidence = append(g.Evidence, domain.EvidenceRef{
			EvidenceID: item.ID, SourceID: item.SourceID, ContentVersion: item.ContentVersion,
		})
	}
	return decision.FilterAttributes([]domain.CandidateGroup{g}, st.snap.Config().Decision.AttributeAllowlist)
}

// assembleReserve 执行 reserve 阶段（组装）。
func (e *Engine) assembleReserve(root *deadline.Root, st *state) (domain.Response, error) {
	if root.Expired(e.Clock.Now()) {
		return e.assembleDegraded(root, st, domain.ReasonDeadlineExceeded)
	}
	stageCtx, cancel := root.Stage(StageReserve, e.Clock.Now(), st.snap.Budget().Reserve.Std())
	defer cancel()
	start := e.Clock.Now()
	// 最终有效证据 = 首轮 + 补检（补检结果已经 CachedProvider 重新过滤）
	final := st.evidence
	final.Evidence = append(append([]domain.Evidence(nil), st.evidence.Evidence...), st.supplement.Evidence...)
	groups := candidateGroups(st)
	// 组装器按请求快照配置构建（Reload 后新请求即用新策略）
	assembler := e.Assembler
	if assembler == nil {
		assembler = NewBuilder(st.snap.Config().Answer)
	}
	resp, err := assembler.Assemble(stageCtx, AssembleInput{
		Request:       st.req,
		Evidence:      final,
		Groups:        groups,
		Decision:      st.decision,
		Degraded:      st.degraded,
		Reasons:       st.reasons,
		TraceID:       st.trace,
		ConfigVersion: st.snap.Version(),
	})
	elapsed := e.Clock.Now().Sub(start)
	remaining := time.Duration(0)
	if d, ok := stageCtx.Deadline(); ok {
		remaining = d.Sub(e.Clock.Now())
	}
	outcome := "ok"
	if err != nil {
		outcome = "error"
	}
	e.Bus.Emit(obs.StageEndEvent{
		Meta: withAt(st.meta, e.Clock.Now()), Stage: StageReserve, Provider: "internal",
		Outcome: outcome, Duration: elapsed, Remaining: remaining,
	})
	return resp, err
}

// assembleDegraded 安全降级路径（root 已到期）：不再调用任何 provider，
// 直接组装 abstain + reason（S5 接管具体动作）。
func (e *Engine) assembleDegraded(root *deadline.Root, st *state, reason string) (domain.Response, error) {
	st.degraded = true
	st.reasons = append(st.reasons, reason)
	st.decision = domain.Decision{}
	assembler := e.Assembler
	if assembler == nil {
		assembler = NewBuilder(st.snap.Config().Answer)
	}
	return assembler.Assemble(context.WithoutCancel(root.Ctx()), AssembleInput{
		Request:       st.req,
		Evidence:      st.evidence,
		Degraded:      true,
		Reasons:       st.reasons,
		TraceID:       st.trace,
		ConfigVersion: st.snap.Version(),
	})
}

// emitProviderError 发 provider 错误事件（error_class 有界枚举）。
func (e *Engine) emitProviderError(stage, op string, err error, st *state) {
	class := "error"
	switch {
	case isDeadlineErr(err):
		class = "timeout"
	case errors.Is(err, context.Canceled):
		class = "canceled"
	case errors.Is(err, domain.ErrProviderContract):
		class = "contract"
	}
	e.Bus.Emit(obs.ProviderErrorEvent{
		Meta: withAt(st.meta, e.Clock.Now()), Provider: providerOfStage(stage), Operation: op, ErrorClass: class,
	})
}

func providerOfStage(stage string) string {
	if stage == StageRejudge {
		return "decision"
	}
	return "evidence"
}

func isDeadlineErr(err error) bool {
	return errors.Is(err, domain.ErrDeadlineExceeded) ||
		errors.Is(err, context.DeadlineExceeded)
}

func outcomeOf(resp domain.Response, err error) string {
	switch {
	case err == nil:
		if resp.Degraded {
			return "degraded"
		}
		return "ok"
	case errors.Is(err, domain.ErrInvalidRequest):
		return "invalid_request"
	case errors.Is(err, domain.ErrDeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, domain.ErrEvidenceUnavailable):
		return "evidence_unavailable"
	case errors.Is(err, domain.ErrDecisionUnavailable):
		return "decision_unavailable"
	default:
		return "error"
	}
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

func withAt(m obs.Meta, at time.Time) obs.Meta {
	m.At = at
	return m
}
