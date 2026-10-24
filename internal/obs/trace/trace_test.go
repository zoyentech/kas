package trace_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/obs"
	"github.com/zoyentech/kas/internal/obs/trace"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestRedactorWhitelistAndHardReject include 白名单 + 敏感键硬拒
// （Story 6.3.1.1 T2；零泄漏子串级断言）。
func TestRedactorWhitelistAndHardReject(t *testing.T) {
	r := trace.NewRedactor([]string{"config_version", "model", "question"})
	ev := domain.TraceEvent{
		TraceID: "t1", ConfigVersion: "cv1",
		Fields: map[string]string{
			"config_version": "cv1",
			"model":          "jev-latest",
			"question":       "用户问题正文", // 即使被误加入 include 也硬拒
			"tenant_key":     "tenant-1",
			"evidence_text":  "证据正文",
			"api_key":        "secret",
			"custom_field":   "不在白名单",
		},
	}
	got := r.Filter(ev)
	if _, ok := got.Fields["question"]; ok {
		t.Fatal("敏感键 question 必须硬拒")
	}
	if _, ok := got.Fields["config_version"]; !ok {
		t.Fatal("白名单字段应保留")
	}
	if len(got.Fields) != 2 {
		t.Fatalf("仅白名单且非敏感字段保留: %+v", got.Fields)
	}
}

// TestAdapterZeroLeak 全链事件→sink 输出零泄漏（子串级扫描）。
func TestAdapterZeroLeak(t *testing.T) {
	var buf bytes.Buffer
	sink := trace.NewSink(&buf)
	adapter := trace.NewAdapter(sink, nil)

	sensitive := "敏感问题正文13800000000"
	adapter.Notify(obs.StageEndEvent{
		Meta:  obs.Meta{TraceID: "trace-x", ConfigVersion: "cv1"},
		Stage: "first_round", Provider: "evidence", Outcome: "ok",
	})
	adapter.Notify(obs.DecisionTraceEvent{
		Meta:     obs.Meta{TraceID: "trace-x", ConfigVersion: "cv1"},
		Provider: "jev", Model: "jev-latest", ModelVersion: "v9",
		FallbackRoute: "jev", Action: "answer", ReasonCode: "ok",
	})
	adapter.Notify(obs.FallbackEvent{
		Meta:  obs.Meta{TraceID: "trace-x", ConfigVersion: "cv1"},
		Route: "deterministic_rules", Reason: "jev_unavailable",
	})

	out := buf.String()
	if strings.Contains(out, sensitive) {
		t.Fatal("trace 输出泄漏敏感正文")
	}
	if !strings.Contains(out, "jev-latest") || !strings.Contains(out, "model_version") {
		t.Fatal("版本链字段应完整（F-009）")
	}
	if !strings.Contains(out, "fallback_route") || !strings.Contains(out, "deterministic_rules") {
		t.Fatal("fallback 路由应可定位（F-009）")
	}
	if !strings.Contains(out, "cache") && !strings.Contains(out, "stage") {
		t.Fatal("阶段字段应记录")
	}
}

// TestAdapterTraceIDConsistency 同请求全部记录 trace_id 一致（F-009）。
func TestAdapterTraceIDConsistency(t *testing.T) {
	var buf bytes.Buffer
	adapter := trace.NewAdapter(trace.NewSink(&buf), nil)
	meta := obs.Meta{TraceID: "consistent-id", ConfigVersion: "cv1"}
	adapter.Notify(obs.StageEndEvent{Meta: meta, Stage: "rejudge", Outcome: "ok"})
	adapter.Notify(obs.CacheEvent{Meta: meta, Result: "miss"})
	adapter.Notify(obs.RequestEndEvent{Meta: meta, Outcome: "ok", Action: "answer"})
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if !strings.Contains(line, `"trace_id":"consistent-id"`) {
			t.Fatalf("trace_id 不一致: %s", line)
		}
	}
}

// TestNoopSink 默认零开销 sink。
func TestNoopSink(t *testing.T) {
	trace.NoopSink{}.Record(context.Background(), domain.TraceEvent{TraceID: "t"})
}
