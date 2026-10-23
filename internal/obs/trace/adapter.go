package trace

import (
	"context"
	"strconv"

	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/obs"
)

// Adapter 把内部事件流转换为 TraceSink 记录（订阅事件总线）。
type Adapter struct {
	Sink     domain.TraceSink
	Redactor *Redactor
}

// NewAdapter 构建事件→trace 适配器。
func NewAdapter(sink domain.TraceSink, redactor *Redactor) *Adapter {
	if redactor == nil {
		redactor = NewRedactor(nil)
	}
	return &Adapter{Sink: sink, Redactor: redactor}
}

// Notify 实现 obs.Subscriber。
func (a *Adapter) Notify(ev obs.Event) {
	if a.Sink == nil {
		return
	}
	var te domain.TraceEvent
	switch e := ev.(type) {
	case obs.StageEndEvent:
		te = domain.TraceEvent{TraceID: e.TraceID, ConfigVersion: e.ConfigVersion, Fields: map[string]string{
			"stage": e.Stage, "stage_outcome": e.Outcome, "provider": e.Provider,
			"duration_ms":  strconv.FormatInt(e.Duration.Milliseconds(), 10),
			"remaining_ms": strconv.FormatInt(e.Remaining.Milliseconds(), 10),
		}}
	case obs.CacheEvent:
		te = domain.TraceEvent{TraceID: e.TraceID, ConfigVersion: e.ConfigVersion, Fields: map[string]string{
			"cache_result": e.Result,
		}}
	case obs.FallbackEvent:
		te = domain.TraceEvent{TraceID: e.TraceID, ConfigVersion: e.ConfigVersion, Fields: map[string]string{
			"fallback_route": e.Route, "reason_code": e.Reason,
		}}
	case obs.FilterEvent:
		te = domain.TraceEvent{TraceID: e.TraceID, ConfigVersion: e.ConfigVersion, Fields: map[string]string{
			"filtered_count": strconv.Itoa(e.Count), "reason_code": e.Reason,
		}}
	case obs.DecisionTraceEvent:
		te = domain.TraceEvent{TraceID: e.TraceID, ConfigVersion: e.ConfigVersion, Fields: map[string]string{
			"decision_provider": e.Provider, "model": e.Model, "model_version": e.ModelVersion,
			"fallback_route": e.FallbackRoute, "action": e.Action, "reason_code": e.ReasonCode,
		}}
	case obs.RequestEndEvent:
		te = domain.TraceEvent{TraceID: e.TraceID, ConfigVersion: e.ConfigVersion, Fields: map[string]string{
			"outcome": e.Outcome, "action": e.Action,
		}}
	default:
		return
	}
	a.Sink.Record(context.Background(), a.Redactor.Filter(te))
}
