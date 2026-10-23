// Package metrics 实现 Prometheus collector（F-006；docs/design/04 §5 全表）。
// 订阅内部事件总线，无侵入采集；label 严格限 allowlist——label 名单由代码
// 固定（与 docs/04 §5 表逐行一致），配置 allowlist 只能收窄；值全部来自
// 受控枚举。禁止 question/身份明文/Evidence ID/trace ID/错误原文/
// config_version/模型版本进 label。KAS 不创建 /metrics listener（宿主提供）。
package metrics

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/obs"
)

// Collector 是事件→指标采集器（实现 obs.Subscriber）。
type Collector struct {
	cfg config.MetricsConfig
	reg prometheus.Registerer

	// 各 vec 实际生效的 label 名单（allowlist 过滤后；Notify 按此组装值，
	// 杜绝 label 数量不匹配 panic）
	reqDurLabels  []string
	reqCntLabels  []string
	stageLabels   []string
	stageDlLabels []string
	deadlineLbls  []string
	cacheLabels   []string
	evictLabels   []string
	filterLabels  []string
	errLabels     []string
	fbLabels      []string
	healthLabels  []string

	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	stageDuration    *prometheus.HistogramVec
	stageDeadline    *prometheus.CounterVec
	deadlineTotal    *prometheus.CounterVec
	cacheTotal       *prometheus.CounterVec
	cacheEvictions   *prometheus.CounterVec
	evidenceFiltered *prometheus.CounterVec
	providerErrors   *prometheus.CounterVec
	decisionFallback *prometheus.CounterVec
	healthStatus     *prometheus.GaugeVec
}

// New 构建 collector；namespace 取配置（默认 kas）。
func New(cfg config.MetricsConfig) *Collector {
	ns := cfg.Namespace
	if ns == "" {
		ns = "kas"
	}
	c := &Collector{cfg: cfg}
	c.reqCntLabels = labelsFiltered(cfg, "operation", "outcome", "action")
	c.reqDurLabels = labelsFiltered(cfg, "operation", "outcome")
	c.stageLabels = labelsFiltered(cfg, "stage", "provider", "outcome")
	c.stageDlLabels = labelsFiltered(cfg, "stage")
	c.deadlineLbls = labelsFiltered(cfg, "deadline_stage")
	c.cacheLabels = labelsFiltered(cfg, "result")
	c.evictLabels = labelsFiltered(cfg, "reason")
	c.filterLabels = labelsFiltered(cfg, "reason")
	c.errLabels = labelsFiltered(cfg, "provider", "operation", "error_class")
	c.fbLabels = labelsFiltered(cfg, "route", "reason")
	c.healthLabels = labelsFiltered(cfg, "component", "status")

	c.requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_requests_total", Help: "请求总数（label: operation/outcome/action；docs/design/04 §5）",
	}, c.reqCntLabels)
	c.requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: ns + "_request_duration_seconds", Help: "请求耗时分布（label: operation/outcome）",
		Buckets: prometheus.DefBuckets,
	}, c.reqDurLabels)
	c.stageDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: ns + "_stage_duration_seconds", Help: "阶段耗时分布（label: stage/provider/outcome）",
		Buckets: prometheus.DefBuckets,
	}, c.stageLabels)
	c.stageDeadline = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_stage_deadline_exceeded_total", Help: "阶段 deadline 超时计数（label: stage）",
	}, c.stageDlLabels)
	c.deadlineTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_deadline_exceeded_total", Help: "deadline 超时计数（label: deadline_stage）",
	}, c.deadlineLbls)
	c.cacheTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_evidence_cache_total", Help: "Evidence 缓存读写结果（label: result=hit/miss/bypass/error）",
	}, c.cacheLabels)
	c.cacheEvictions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_evidence_cache_evictions_total", Help: "缓存淘汰计数（label: reason）",
	}, c.evictLabels)
	c.evidenceFiltered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_evidence_filtered_total", Help: "Evidence 过滤计数（label: reason=expired/future/unauthorized/unknown_validity）",
	}, c.filterLabels)
	c.providerErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_provider_errors_total", Help: "provider 错误计数（label: provider/operation/error_class）",
	}, c.errLabels)
	c.decisionFallback = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: ns + "_decision_fallback_total", Help: "决策降级路由计数（label: route/reason）",
	}, c.fbLabels)
	c.healthStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: ns + "_health_status", Help: "组件健康状态（label: component/status；值 1=当前状态）",
	}, c.healthLabels)
	return c
}

// labelsFiltered 按 label_allowlist 过滤 label 名：不在名单的 label 从指标
// 定义中移除（基数与合规双保险）。
func labelsFiltered(cfg config.MetricsConfig, want ...string) []string {
	if len(cfg.LabelAllowlist) == 0 {
		return want
	}
	allowed := map[string]bool{}
	for _, l := range cfg.LabelAllowlist {
		allowed[l] = true
	}
	var out []string
	for _, l := range want {
		if allowed[l] {
			out = append(out, l)
		}
	}
	return out
}

// pick 从 (name,value) 对中按 vec 的 label 名单选取值并按名单排序返回。
// kv 形如 "stage", stage, "deadline_stage", ds。
func pick(names []string, kv ...string) []string {
	vals := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		vals[kv[i]] = kv[i+1]
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, vals[n])
	}
	return out
}

// collectors 返回全部已注册 collector（注册/注销共用）。
func (c *Collector) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		c.requestsTotal, c.requestDuration, c.stageDuration, c.stageDeadline,
		c.deadlineTotal, c.cacheTotal, c.cacheEvictions, c.evidenceFiltered,
		c.providerErrors, c.decisionFallback, c.healthStatus,
	}
}

// RegisterTo 注册到宿主 registry；同名冲突返回错误（映射 ErrInvalidConfig
// 由调用方包装）。
func (c *Collector) RegisterTo(reg prometheus.Registerer) error {
	if reg == nil {
		return nil
	}
	c.reg = reg
	for _, col := range c.collectors() {
		if err := reg.Register(col); err != nil {
			return fmt.Errorf("kas: metrics 注册失败（同名冲突需检查宿主 registry）: %w", err)
		}
	}
	return nil
}

// Unregister 注销全部自有 collector（Close 调用）。
func (c *Collector) Unregister() {
	if c.reg == nil {
		return
	}
	for _, col := range c.collectors() {
		c.reg.Unregister(col)
	}
}

// Notify 实现 obs.Subscriber：事件→指标映射（label 值按 vec 名单组装）。
func (c *Collector) Notify(ev obs.Event) {
	switch e := ev.(type) {
	case obs.RequestEndEvent:
		c.requestsTotal.WithLabelValues(pick(c.reqCntLabels,
			"operation", "answer", "outcome", e.Outcome, "action", e.Action)...).Inc()
		c.requestDuration.WithLabelValues(pick(c.reqDurLabels,
			"operation", "answer", "outcome", e.Outcome)...).Observe(e.Duration.Seconds())
	case obs.StageEndEvent:
		c.stageDuration.WithLabelValues(pick(c.stageLabels,
			"stage", e.Stage, "provider", e.Provider, "outcome", e.Outcome)...).Observe(e.Duration.Seconds())
	case obs.StageDeadlineEvent:
		c.stageDeadline.WithLabelValues(pick(c.stageDlLabels, "stage", e.Stage)...).Inc()
		c.deadlineTotal.WithLabelValues(pick(c.deadlineLbls, "deadline_stage", e.DeadlineStage)...).Inc()
	case obs.CacheEvent:
		c.cacheTotal.WithLabelValues(pick(c.cacheLabels, "result", e.Result)...).Inc()
	case obs.CacheEvictionEvent:
		c.cacheEvictions.WithLabelValues(pick(c.evictLabels, "reason", e.Reason)...).Inc()
	case obs.FilterEvent:
		c.evidenceFiltered.WithLabelValues(pick(c.filterLabels, "reason", e.Reason)...).Add(float64(e.Count))
	case obs.ProviderErrorEvent:
		c.providerErrors.WithLabelValues(pick(c.errLabels,
			"provider", e.Provider, "operation", e.Operation, "error_class", e.ErrorClass)...).Inc()
	case obs.FallbackEvent:
		c.decisionFallback.WithLabelValues(pick(c.fbLabels,
			"route", e.Route, "reason", e.Reason)...).Inc()
	case obs.HealthEvent:
		c.healthStatus.WithLabelValues(pick(c.healthLabels,
			"component", e.Component, "status", e.Status)...).Set(1)
	}
}

// CheckLabelValues 是禁入 label 值的机械防护（docs/04 §5 禁令）：
// 采集路径只产生受控枚举值；本函数供防护测试与 CI 扫描注入使用，
// 返回命中禁令的值（question/身份/Evidence ID/trace ID/config_version/
// 模型版本/自由文本）。
func CheckLabelValues(values ...string) []string {
	var bad []string
	for _, v := range values {
		lv := strings.ToLower(v)
		for _, kw := range []string{"question", "tenant", "principal", "token", "evidence_", "trace-", "config_version", "model_version"} {
			if strings.Contains(lv, kw) {
				bad = append(bad, v)
				break
			}
		}
	}
	return bad
}
