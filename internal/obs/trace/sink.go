// Package trace 实现 TraceSink 与脱敏（F-009；docs/design/04 §6）：
// 字段级 include 白名单 + 敏感键硬拒；默认结构化 sink（JSON 行），
// 可注入自定义 sink。KAS 不落盘、不外发——输出目标由宿主注入。
package trace

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/zoyentech/kas/internal/domain"
)

// Sink 是默认结构化 sink：JSON 行写入 w（宿主控制 w 的去向）。
type Sink struct {
	mu sync.Mutex
	w  io.Writer
}

// NewSink 构建结构化 sink。
func NewSink(w io.Writer) *Sink { return &Sink{w: w} }

// Record 写入一条 JSON 行（已由 Redactor 过滤）。
func (s *Sink) Record(ctx context.Context, event domain.TraceEvent) {
	b, err := json.Marshal(event)
	if err != nil {
		return // 字段值异常时丢弃该条（不中断请求路径）
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.w.Write(append(b, '\n'))
}

// NoopSink 丢弃全部记录（默认零开销）。
type NoopSink struct{}

// Record no-op。
func (NoopSink) Record(ctx context.Context, event domain.TraceEvent) {}
