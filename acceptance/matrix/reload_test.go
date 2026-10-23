package matrix

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	kas "github.com/kas-runtime/kas"
	"github.com/kas-runtime/kas/internal/domain"
)

// 矩阵行 8：Reload 并发——在途 response 为旧版，新请求为新版。
func TestRowReloadConcurrent(t *testing.T) {
	release := make(chan struct{})
	ev := &gateEv{called: make(chan struct{}), release: release}
	c := newClient(t, ev, nil, nil)
	v0 := c.ConfigVersion()

	type result struct {
		resp kas.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		r, err := c.Answer(context.Background(), req("Reload 并发问题"))
		done <- result{r, err}
	}()
	<-ev.called
	mutated := strings.Replace(baseYAML, "top_k: 4", "top_k: 5", 1)
	if _, err := c.Reload(context.Background(), strings.NewReader(mutated)); err != nil {
		t.Fatal(err)
	}
	if c.ConfigVersion() == v0 {
		t.Fatal("Reload 后版本应变化")
	}
	close(release)
	r := <-done
	if r.err != nil {
		t.Fatal(r.err)
	}
	if r.resp.ConfigVersion != v0 {
		t.Fatalf("在途请求应持旧版 %s，实际 %s", v0, r.resp.ConfigVersion)
	}
	// 新请求读新版
	r2, err := c.Answer(context.Background(), req("Reload 后新请求"))
	if err != nil {
		t.Fatal(err)
	}
	if r2.ConfigVersion == v0 {
		t.Fatal("新请求应读新版本")
	}
}

// gateEv 慢检索（等待 release）。
type gateEv struct {
	called  chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *gateEv) Search(ctx context.Context, in kas.EvidenceQuery) (kas.EvidenceResult, error) {
	g.once.Do(func() { close(g.called) })
	select {
	case <-g.release:
		until := time.Now().Add(time.Hour)
		return kas.EvidenceResult{Evidence: []kas.Evidence{{ID: "e1", SourceID: "s", ValidUntil: &until}}}, nil
	case <-ctx.Done():
		return kas.EvidenceResult{}, ctx.Err()
	}
}

func (g *gateEv) Health(ctx context.Context) kas.ComponentHealth {
	return kas.ComponentHealth{Status: kas.ComponentHealthy}
}

// 矩阵行 9：Reload 配置无效——返回错误，当前 config_version 不变。
func TestRowReloadInvalid(t *testing.T) {
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
	c := newClient(t, ev, nil, nil)
	v0 := c.ConfigVersion()

	bad := strings.Replace(baseYAML, "reserve: 60ms, min_reserve: 40ms", "reserve: 61ms, min_reserve: 40ms", 1)
	if _, err := c.Reload(context.Background(), strings.NewReader(bad)); err == nil {
		t.Fatal("破坏预算等式的 Reload 必须失败")
	}
	if c.ConfigVersion() != v0 {
		t.Fatalf("无效 Reload 后版本变化: %s -> %s", v0, c.ConfigVersion())
	}
	resp, err := c.Answer(context.Background(), req("无效 Reload 后请求"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.ConfigVersion != v0 {
		t.Fatalf("行为仍按旧版本: %+v", resp)
	}
}
