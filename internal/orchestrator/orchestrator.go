package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/evidence"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/runtime"
)

// Clock 抽象时间源（生产用系统时钟；测试注入 fake clock）。
type Clock interface {
	Now() time.Time
}

// SystemClock 是生产时钟。
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// PreliminaryOutcome 是初判结果（docs/design/01 §4 第 4 步）。
type PreliminaryOutcome struct {
	Kind   PreliminaryKind
	Reason string // 受控 reason code
}

// PreliminaryKind 是初判路径枚举。
type PreliminaryKind string

const (
	PreliminaryContinue PreliminaryKind = "continue"
	PreliminaryClarify  PreliminaryKind = "ask_clarification"
	PreliminaryAbstain  PreliminaryKind = "abstain"
)

// PreliminaryInput 是初判输入。
type PreliminaryInput struct {
	Request       domain.Request
	ConfigVersion string
}

// PreliminaryJudge 是初判阶段挂载点（S5 规则引擎替换默认实现）。
type PreliminaryJudge interface {
	Judge(ctx context.Context, in PreliminaryInput) (PreliminaryOutcome, error)
}

// AssembleInput 是回答组装输入（reserve 阶段）。
type AssembleInput struct {
	Request       domain.Request
	Evidence      domain.EvidenceResult // 最终有效证据（首轮 + 补检重滤后合并）
	Groups        []domain.CandidateGroup
	Decision      domain.Decision
	Degraded      bool
	Reasons       []string
	TraceID       string
	ConfigVersion string
}

// Assembler 是回答组装挂载点（S5 citation 安全校验替换默认实现）。
type Assembler interface {
	Assemble(ctx context.Context, in AssembleInput) (domain.Response, error)
}

// Engine 是请求编排器（Story 3.3.1.1）：校验 Request → 读 snapshot →
// 固定 config_version → 五阶段流水线 → 组装 Response。
type Engine struct {
	Manager     *runtime.Manager
	Bus         *obs.Bus
	Clock       Clock
	Preliminary PreliminaryJudge
	Assembler   Assembler
	Evidence    domain.EvidenceProvider
	Decision    domain.DecisionProvider
	// Cache 是 Evidence 缓存实例（nil = 缓存禁用；检索仍经过滤路径）。
	Cache domain.EvidenceCache
	// Perm 是权限判定器（nil = 不做权限过滤；由装配方按 O-7 注入）。
	Perm    evidence.Checker
	TraceID func() string
}

// NewEngine 构建编排器；nil 字段落默认值（初判放行、最小组装器、系统时钟）。
func NewEngine(m *runtime.Manager, bus *obs.Bus, ev domain.EvidenceProvider, dec domain.DecisionProvider) *Engine {
	e := &Engine{
		Manager:     m,
		Bus:         bus,
		Clock:       SystemClock{},
		Preliminary: DefaultPreliminary{},
		// Assembler 留 nil：reserve 阶段按请求快照构建 Builder（引用安全闸）
		Evidence: ev,
		Decision: dec,
		TraceID:  NewTraceID,
	}
	return e
}

// NewTraceID 生成 128bit 随机 hex（32 字符）trace ID。
func NewTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 加密源失败时退化为时间戳：trace ID 仅是关联键，不承担安全语义
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000000000")))[:32]
	}
	return hex.EncodeToString(b[:])
}
