# KAS v1.2 Go SDK API 草案

领域接口草案，不绑定 HTTP 框架、Prometheus server 或 Jev 协议。实现评审时可调整命名与错误码。

## 1. 初始化与配置

~~~go
type Config struct {
    SchemaVersion int
    Timeouts       TimeoutBudget
    Evidence       EvidenceConfig
    Decision       DecisionConfig
    Answer         AnswerConfig
}

type TimeoutBudget struct {
    Total, Preliminary, FirstRound, Rejudge, Supplement, Reserve time.Duration
}

func LoadConfig(r io.Reader) (Config, error)
func LoadConfigFile(path string) (Config, error)
func ValidateConfig(cfg Config) error
func New(cfg Config, deps Dependencies, opts ...Option) (*Client, error)

type Dependencies struct {
    EvidenceProvider EvidenceProvider
    DecisionProvider DecisionProvider
    Cache            EvidenceCache
    HealthChecks     HealthChecks
    TraceSink        TraceSink
}
~~~

New 同步校验预算等式、duration 溢出、必要依赖、cache 容量、有效期策略和 provider 参数。密钥由宿主 secret manager/env 注入，不能写入 YAML 或错误文本。

## 2. Request / Response

~~~go
type Request struct {
    Question     string
    Principal    PrincipalScope
    Knowledge    KnowledgeScope
    Conversation ConversationContext
    RequestID    string
}

type PrincipalScope struct {
    TenantKey          string // 不透明内部标识，不进入日志/指标
    ScopeFingerprint   string // 宿主基于有效权限范围生成
    AuthorizationEpoch string // 权限变化时变更
}

type KnowledgeScope struct {
    Namespace        string
    CorpusGeneration string
}

type Response struct {
    Action        ActionType
    Answer        string
    Citations     []Citation
    Confidence    *float64
    TraceID       string
    ConfigVersion string
    Degraded      bool
    Degradation   []string // 受控 reason code
}

type ActionType string
const (
    ActionAnswer ActionType = "answer"
    ActionSupplement ActionType = "supplement"
    ActionClarify ActionType = "ask_clarification"
    ActionAbstain ActionType = "abstain"
)
~~~

宿主先完成认证授权；KAS 不从 question 推导 scope。answer 必须至少有一条有效引用，除非业务批准无 Evidence 的纯对话模式。

## 3. DecisionProvider

~~~go
type CandidateGroup struct {
    ID         string
    Evidence   []EvidenceRef
    Score      float64
    Attributes map[string]string // 仅配置白名单字段
}

type DecisionInput struct {
    Question      string
    Groups        []CandidateGroup
    ValidAt       time.Time
    ConfigVersion string
    ModelHint     string
}

type Decision struct {
    Action         ActionType
    TargetGroupIDs []string
    NeedSupplement bool
    ReasonCode     string
    Confidence     float64
    Provider       string
    Model          string
    ModelVersion   string
    Degraded       bool
}

type DecisionProvider interface {
    Decide(ctx context.Context, in DecisionInput) (Decision, error)
    Health(ctx context.Context) ComponentHealth
}
~~~

每个 target group ID 必须来自输入；action 必须属于枚举；NaN/Inf confidence 或未知值按 provider contract error 拒绝。Jev 私有协议只存在于 Jev adapter。Jev 默认最多调用一次，以免重试耗尽补检/余量预算。

## 4. Evidence 合约

~~~go
type EvidenceProvider interface {
    Search(ctx context.Context, in EvidenceQuery) (EvidenceResult, error)
    Health(ctx context.Context) ComponentHealth
}

type Evidence struct {
    ID             string
    SourceID       string
    Title          string
    Excerpt        string
    ValidFrom      *time.Time
    ValidUntil     *time.Time
    ObservedAt     time.Time
    ContentVersion string
    PermissionTags []string
    Score          float64
}
~~~

有效期优先使用来源显式时间；缺失时只能按批准的 source max-age 推导，策略缺失则 fail-closed。每轮补检结果也要重新授权和有效期过滤。

## 5. Client / Health / Reload

~~~go
type Client struct { /* 私有字段，并发安全 */ }

func (c *Client) Answer(ctx context.Context, req Request) (Response, error)
func (c *Client) Health(ctx context.Context) HealthReport
func (c *Client) Reload(ctx context.Context, r io.Reader) (string, error)
func (c *Client) Close() error
~~~

Reload 依次解析、Validate、准备候选 runtime/provider、计算脱敏 config version、原子替换。任一步失败均保持原 snapshot。在途请求持有开始时读取的 snapshot，新请求读替换后的版本。

Health 总状态为 healthy | degraded | not_ready。必需 EvidenceProvider 故障为 not_ready；可选 Jev 故障且安全 fallback 可用时为 degraded。Health 不执行用户查询或模型生成，使用短独立 timeout。

## 6. 扩展点与错误

~~~go
type EvidenceCache interface {
    Get(ctx context.Context, key CacheKey) (EvidenceResult, bool, error)
    Put(ctx context.Context, key CacheKey, value EvidenceResult, ttl time.Duration) error
    DeleteNamespace(ctx context.Context, namespace string) error
}

type TraceSink interface {
    Record(ctx context.Context, event TraceEvent)
}
~~~

Prometheus collector 注册到宿主 registry；KAS 不开 listener。错误至少区分 ErrInvalidConfig、ErrInvalidRequest、ErrDeadlineExceeded、ErrEvidenceUnavailable、ErrDecisionUnavailable、ErrProviderContract 和 ErrClosed。错误原文不直接返回用户。

Client 方法并发安全。Close 停止新请求并关闭 SDK 自有资源；宿主传入且共享的 HTTP client、registry 等归宿主管理。
