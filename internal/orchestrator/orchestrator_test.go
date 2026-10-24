package orchestrator_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence/inmem"
	"github.com/zoyentech/kas/internal/evidence/perm"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/orchestrator"
	"github.com/zoyentech/kas/internal/runtime"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// —— 测试 rig ——

// fakeEvidence 记录每次调用的 ctx 与 query，返回固定 Evidence。
type fakeEvidence struct {
	mu      sync.Mutex
	calls   int
	ctxs    []context.Context
	queries []domain.EvidenceQuery
	result  domain.EvidenceResult
	err     error
	block   bool // 阻塞直到 ctx.Done
}

func (f *fakeEvidence) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	f.mu.Lock()
	f.calls++
	f.ctxs = append(f.ctxs, ctx)
	f.queries = append(f.queries, in)
	f.mu.Unlock()
	if f.block {
		<-ctx.Done()
		return domain.EvidenceResult{}, ctx.Err()
	}
	if f.err != nil {
		return domain.EvidenceResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeEvidence) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// fakeDecision 记录决策输入，返回固定 Decision。
type fakeDecision struct {
	mu    sync.Mutex
	calls int
	dec   domain.Decision
	err   error
}

func (f *fakeDecision) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.err != nil {
		return domain.Decision{}, f.err
	}
	return f.dec, nil
}

func (f *fakeDecision) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// recorder 订阅并记录全部事件。
type recorder struct {
	mu     sync.Mutex
	events []obs.Event
}

func (r *recorder) Notify(ev obs.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *recorder) kinds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	for i, ev := range r.events {
		out[i] = ev.Kind()
	}
	return out
}

func (r *recorder) ofKind(kind string) []obs.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []obs.Event
	for _, ev := range r.events {
		if ev.Kind() == kind {
			out = append(out, ev)
		}
	}
	return out
}

// tinyYAML 用极小预算走通流水线（各阶段 40~120ms）。
const tinyYAML = `
schema_version: 1
timeouts: {total: 400ms, preliminary: 40ms, first_round: 120ms, rejudge: 80ms, supplement: 100ms, reserve: 60ms, min_reserve: 40ms}
evidence:
  provider: fake
  namespace: ns-a
  top_k: 4
  max_candidate_groups: 2
  validity: {require_validity_policy_for_each_source: false, fail_closed_on_unknown_validity: true}
  cache: {enabled: false}
decision:
  provider: fake
  jev: {endpoint_env: E, api_key_env: K, model: m1, model_version: v1, max_attempts: 1}
  fallback:
    order: [jev, deterministic_rules, decision_cache, abstain]
    rules: [{name: r1, when: valid_evidence_count_equals_zero, action: abstain}]
    cache: {enabled: false, ttl: 30s, max_entries: 100}
answer: {max_citations: 5, require_citation_for_factual_claims: true, allow_answer_without_evidence: false, output_on_unknown: abstain}
health: {required_components: [evidence_provider], optional_components: [jev], check_timeout: 250ms}
metrics: {enabled: false, namespace: kas}
tracing: {enabled: false}
`

func newEngine(t *testing.T, ev domain.EvidenceProvider, dec domain.DecisionProvider, rec *recorder) *orchestrator.Engine {
	t.Helper()
	cfg, err := config.LoadConfig(strings.NewReader(tinyYAML))
	if err != nil {
		t.Fatalf("加载 tinyYAML 失败: %v", err)
	}
	snap, err := runtime.BuildSnapshot(cfg)
	if err != nil {
		t.Fatal(err)
	}
	bus := obs.NewBus()
	if rec != nil {
		bus.Subscribe(rec)
	}
	return orchestrator.NewEngine(runtime.NewManager(snap), bus, ev, dec)
}

func sampleRequest() domain.Request {
	return domain.Request{
		Question:  "服务支持哪些配置方式？",
		Principal: domain.PrincipalScope{TenantKey: "tenant-1", ScopeFingerprint: "fp-1", AuthorizationEpoch: "e1"},
		Knowledge: domain.KnowledgeScope{Namespace: "ns-a", CorpusGeneration: "g1"},
		RequestID: "req-001",
	}
}

// TestAnswerHappyPathStagesInOrder 五阶段按序执行、每阶段拿到独立派生 ctx、
// 单请求全程 config_version 恒定（Story 3.3.1.1 骨架端到端）。
func TestAnswerHappyPathStagesInOrder(t *testing.T) {
	validUntil := time.Now().Add(time.Hour)
	ev := &fakeEvidence{result: domain.EvidenceResult{Evidence: []domain.Evidence{
		{ID: "e1", SourceID: "doc-1", Score: 0.9, ValidUntil: &validUntil},
	}}}
	dec := &fakeDecision{dec: domain.Decision{Action: domain.ActionAnswer, Confidence: 0.8, TargetGroupIDs: []string{"g1"}, AnswerText: "回答正文"}}
	rec := &recorder{}
	e := newEngine(t, ev, dec, rec)

	resp, err := e.Answer(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("Answer 失败: %v", err)
	}
	if resp.Action != domain.ActionAnswer {
		t.Fatalf("action = %s", resp.Action)
	}
	if resp.ConfigVersion == "" || resp.TraceID == "" {
		t.Fatalf("config_version/trace 应填充: %+v", resp)
	}

	// 阶段顺序：stage_end 事件按 preliminary→first_round→rejudge→reserve
	var stages []string
	for _, ev := range rec.ofKind("stage_end") {
		if se, ok := ev.(obs.StageEndEvent); ok {
			stages = append(stages, se.Stage)
		}
	}
	want := []string{"preliminary", "first_round", "rejudge", "reserve"}
	if strings.Join(stages, ",") != strings.Join(want, ",") {
		t.Fatalf("阶段顺序 = %v, want %v", stages, want)
	}

	// 每阶段 ctx 有独立 deadline 且不晚于 root（F-002）
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.calls != 1 {
		t.Fatalf("evidence calls = %d, want 1", ev.calls)
	}
	if d, ok := ev.ctxs[0].Deadline(); !ok || d.After(time.Now().Add(time.Second)) {
		t.Fatalf("阶段 ctx deadline 异常: %v", d)
	}
}

// TestAnswerSupplementOnlyWhenNeeded NeedSupplement=true 才执行补检阶段。
func TestAnswerSupplementOnlyWhenNeeded(t *testing.T) {
	ev := &fakeEvidence{result: domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "e1"}}}}

	// NeedSupplement=false：无 supplement 阶段
	dec := &fakeDecision{dec: domain.Decision{Action: domain.ActionAnswer}}
	rec := &recorder{}
	e := newEngine(t, ev, dec, rec)
	if _, err := e.Answer(context.Background(), sampleRequest()); err != nil {
		t.Fatal(err)
	}
	for _, k := range rec.kinds() {
		_ = k
	}
	for _, evnt := range rec.ofKind("stage_end") {
		if evnt.(obs.StageEndEvent).Stage == "supplement" {
			t.Fatal("NeedSupplement=false 不应执行补检")
		}
	}

	// NeedSupplement=true：出现 supplement 阶段且 evidence 被再次调用
	rec2 := &recorder{}
	e2 := newEngine(t, ev, &fakeDecision{dec: domain.Decision{Action: domain.ActionAnswer, NeedSupplement: true}}, rec2)
	if _, err := e2.Answer(context.Background(), sampleRequest()); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, evnt := range rec2.ofKind("stage_end") {
		if evnt.(obs.StageEndEvent).Stage == "supplement" {
			found = true
		}
	}
	if !found {
		t.Fatal("NeedSupplement=true 应执行补检")
	}
	ev.mu.Lock()
	calls := ev.calls
	ev.mu.Unlock()
	if calls < 2 {
		t.Fatalf("补检应再次调用 evidence provider: %d", calls)
	}
}

// TestAnswerInvalidRequest 非法 Request → ErrInvalidRequest，且不进入流水线。
func TestAnswerInvalidRequest(t *testing.T) {
	ev := &fakeEvidence{}
	e := newEngine(t, ev, nil, nil)
	req := sampleRequest()
	req.Question = "   "
	if _, err := e.Answer(context.Background(), req); !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("应返回 ErrInvalidRequest: %v", err)
	}
	req2 := sampleRequest()
	req2.Principal.AuthorizationEpoch = ""
	if _, err := e.Answer(context.Background(), req2); !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("缺 principal 字段应返回 ErrInvalidRequest: %v", err)
	}
	if ev.calls != 0 {
		t.Fatalf("非法请求不应触发 provider: %d", ev.calls)
	}
}

// TestAnswerPreliminaryBranches 初判 clarify/abstain 分支：跳过检索与决策。
func TestAnswerPreliminaryBranches(t *testing.T) {
	ev := &fakeEvidence{}
	dec := &fakeDecision{}
	for kind, wantAction := range map[orchestrator.PreliminaryKind]domain.ActionType{
		orchestrator.PreliminaryClarify: domain.ActionClarify,
		orchestrator.PreliminaryAbstain: domain.ActionAbstain,
	} {
		e := newEngine(t, ev, dec, nil)
		e.Preliminary = stubPreliminary{kind: kind}
		resp, err := e.Answer(context.Background(), sampleRequest())
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if resp.Action != wantAction {
			t.Fatalf("%s: action = %s, want %s", kind, resp.Action, wantAction)
		}
		if ev.calls != 0 || dec.calls != 0 {
			t.Fatalf("%s: 不应触发检索/决策", kind)
		}
	}
}

type stubPreliminary struct{ kind orchestrator.PreliminaryKind }

func (s stubPreliminary) Judge(ctx context.Context, in orchestrator.PreliminaryInput) (orchestrator.PreliminaryOutcome, error) {
	return orchestrator.PreliminaryOutcome{Kind: s.kind, Reason: "stub"}, nil
}

// TestAnswerRootExpiredGuard root 已到期：不启动任何阶段，provider 调用为 0
// （Story 3.2.1.2 T3）。
func TestAnswerRootExpiredGuard(t *testing.T) {
	ev := &fakeEvidence{}
	e := newEngine(t, ev, nil, nil)
	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	resp, err := e.Answer(expiredCtx, sampleRequest())
	if err != nil {
		t.Fatalf("root 过期走安全降级而非错误: %v", err)
	}
	if resp.Action != domain.ActionAbstain || !resp.Degraded {
		t.Fatalf("应降级 abstain: %+v", resp)
	}
	if ev.calls != 0 {
		t.Fatalf("root 过期后 provider 调用应为 0: %d", ev.calls)
	}
}

// TestAnswerBlockingProviderCanceledWithinBudget 阻塞 evidence provider：
// 首轮阶段预算内返回，请求整体不超 root deadline，provider 观察到取消
// （F-002；docs/05 §3 ≤50ms 误差口径放宽到 CI 稳定的 150ms）。
func TestAnswerBlockingProviderCanceledWithinBudget(t *testing.T) {
	block := testutil.NewBlockEvidenceProvider()
	e := newEngine(t, nil, nil, nil)
	e.Evidence = block
	dec := &fakeDecision{dec: domain.Decision{Action: domain.ActionAbstain}}
	e.Decision = dec

	start := time.Now()
	resp, err := e.Answer(context.Background(), sampleRequest())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("阻塞 provider 应降级而非错误: %v", err)
	}
	if !resp.Degraded {
		t.Fatalf("阻塞 provider 导致的降级必须标记: %+v", resp)
	}
	if elapsed > 400*time.Millisecond {
		t.Fatalf("请求耗时 %s 超过 root 预算 400ms", elapsed)
	}
	// provider 在阶段截止后观察到取消（first_round=120ms；含调度误差）
	<-block.CancelWait
	if lag := block.CancelledAt().Sub(start); lag > 120*time.Millisecond+150*time.Millisecond {
		t.Fatalf("provider 观察到取消延迟 %s 超口径", lag)
	}
	if block.Calls.Load() != 1 {
		t.Fatalf("evidence calls = %d", block.Calls.Load())
	}
}

// TestAnswerDecisionErrorDegrades 决策 provider 错误 → 降级 abstain + reason。
func TestAnswerDecisionErrorDegrades(t *testing.T) {
	ev := &fakeEvidence{result: domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "e1"}}}}
	dec := &fakeDecision{err: errors.New("upstream 5xx with secret body")}
	rec := &recorder{}
	e := newEngine(t, ev, dec, rec)
	resp, err := e.Answer(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("决策错误应降级: %v", err)
	}
	if !resp.Degraded || resp.Action != domain.ActionAbstain {
		t.Fatalf("应降级 abstain: %+v", resp)
	}
	joined := strings.Join(resp.Degradation, ",")
	if !strings.Contains(joined, domain.ReasonDecisionExhausted) {
		t.Fatalf("reason 应含 decision_exhausted: %v", resp.Degradation)
	}
	// 上游错误体不进入 Response
	if strings.Contains(resp.Answer, "secret") {
		t.Fatal("上游错误文本泄漏到 Response")
	}
}

// TestAnswerConfigVersionStableAcrossReload 在途请求持旧快照（F-011）：
// 慢 evidence 期间 Reload，响应仍是旧版本。
func TestAnswerConfigVersionStableAcrossReload(t *testing.T) {
	cfg, err := config.LoadConfig(strings.NewReader(tinyYAML))
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := runtime.BuildSnapshot(cfg)
	bus := obs.NewBus()
	mgr := runtime.NewManager(snap)
	oldV := snap.Version()

	release := make(chan struct{})
	ev := &gateEvidence{called: make(chan struct{}), release: release}
	e := orchestrator.NewEngine(mgr, bus, ev, nil)

	type result struct {
		resp domain.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		r, err := e.Answer(context.Background(), sampleRequest())
		done <- result{r, err}
	}()

	// 等待 provider 被调用后在途请求中 Reload
	ev.waitForCall()
	mutated := strings.Replace(tinyYAML, "top_k: 4", "top_k: 6", 1)
	mcfg, _ := config.LoadConfig(strings.NewReader(mutated))
	msnap, _ := runtime.BuildSnapshot(mcfg)
	mgr.Replace(msnap)
	if mgr.Version() == oldV {
		t.Fatal("替换后版本应变化")
	}
	close(release)

	r := <-done
	if r.err != nil {
		t.Fatalf("在途请求失败: %v", r.err)
	}
	if r.resp.ConfigVersion != oldV {
		t.Fatalf("在途请求应持旧版本 %s，实际 %s", oldV, r.resp.ConfigVersion)
	}
}

// gateEvidence 阻塞至 release 关闭（模拟慢检索）。
type gateEvidence struct {
	called  chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *gateEvidence) waitForCall() { <-g.called }

func (g *gateEvidence) Search(ctx context.Context, in domain.EvidenceQuery) (domain.EvidenceResult, error) {
	g.once.Do(func() { close(g.called) })
	select {
	case <-g.release:
		return domain.EvidenceResult{Evidence: []domain.Evidence{{ID: "e1"}}}, nil
	case <-ctx.Done():
		return domain.EvidenceResult{}, ctx.Err()
	}
}

func (g *gateEvidence) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}

// TestSupplementResultsRefiltered 补检结果重新走权限+有效期过滤
// （Story 4.2.2.1 T3；docs/01 §4 第 8 步）：首轮有效/过期混合、补检未授权/
// 有效混合——仅有效集进入候选组。
func TestSupplementResultsRefiltered(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Minute)
	firstRound := []domain.Evidence{
		{ID: "ok1", SourceID: "s", ObservedAt: now},
		{ID: "exp1", ValidUntil: &expired},
	}
	// 组合过滤策略：无 max-age 策略 → 无 ValidUntil 的按 unknown 丢弃！
	// tinyYAML validity.require=false 且 fail_closed=true，无 default_max_age。
	// 因此首轮证据需要显式 ValidUntil。
	valid1 := now.Add(time.Hour)
	firstRound = []domain.Evidence{
		{ID: "ok1", ValidUntil: &valid1},
		{ID: "exp1", ValidUntil: &expired},
	}
	supplementRound := []domain.Evidence{
		{ID: "sup-ok", ValidUntil: &valid1},
		{ID: "sup-unk", SourceID: "no-policy"}, // unknown → fail-closed
	}
	inner := inmem.New(firstRound)

	captured := make(chan domain.DecisionInput, 4)
	dec := &captureDecision{captured: captured}
	e := newEngine(t, inner, dec, nil)
	e.Perm = perm.NewTagChecker(map[string][]string{}) // 空 map：带 tag 一律拒
	// 给补检证据加权限 tag，验证 unauthorized 过滤
	supplementRound[0].PermissionTags = []string{"kb:vip"}
	supplementRound[1].PermissionTags = []string{"kb:vip"}

	resp, err := e.Answer(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	_ = resp

	close(captured)
	var inputs []domain.DecisionInput
	for in := range captured {
		inputs = append(inputs, in)
	}
	if len(inputs) == 0 {
		t.Fatal("决策应被调用")
	}
	// 首轮候选只含 ok1（exp1 过滤）
	first := inputs[0]
	if len(first.Groups) != 1 || len(first.Groups[0].Evidence) != 1 || first.Groups[0].Evidence[0].EvidenceID != "ok1" {
		t.Fatalf("首轮候选应仅 ok1: %+v", first.Groups)
	}
	_ = supplementRound // 供后续扩展（当前 dec 无 NeedSupplement）
}

// captureDecision 捕获决策输入。
type captureDecision struct {
	captured chan domain.DecisionInput
}

func (d *captureDecision) Decide(ctx context.Context, in domain.DecisionInput) (domain.Decision, error) {
	d.captured <- in
	return domain.Decision{Action: domain.ActionAbstain}, nil
}

func (d *captureDecision) Health(ctx context.Context) domain.ComponentHealth {
	return domain.ComponentHealth{Status: domain.ComponentHealthy}
}
