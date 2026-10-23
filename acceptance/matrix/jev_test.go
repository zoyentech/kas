package matrix

import (
	"context"
	"strings"
	"testing"
	"time"

	kas "github.com/kas-runtime/kas"
	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/decision/jev"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

// 矩阵行 10：Jev 合法响应——action/目标组属于候选。
func TestRowJevValid(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{Body: testutil.ValidJevBody("answer", "g1", 0.9)})
	defer fake.Close()
	ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
	t.Setenv("KAS_MTX_JEV_ENDPOINT", fake.URL())
	t.Setenv("KAS_MTX_JEV_API_KEY", "k")
	cfg, err := kas.LoadConfig(strings.NewReader(baseYAML))
	if err != nil {
		t.Fatal(err)
	}
	c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: ev})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	resp, err := c.Answer(context.Background(), req("Jev 正常问题"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Action != kas.ActionAnswer || resp.Degraded {
		t.Fatalf("Jev 合法响应应 answer 且未降级: %+v", resp)
	}
	if len(resp.Citations) != 1 {
		t.Fatalf("引用应来自候选: %+v", resp.Citations)
	}
}

// 矩阵行 11+12：Jev timeout/5xx/非法响应 → 按序降级，不超 root deadline。
func TestRowJevFaults(t *testing.T) {
	runWith := func(t *testing.T, script testutil.JevScript) kas.Response {
		t.Helper()
		fake := testutil.NewJevFakeServer(script)
		defer fake.Close()
		ev := &matrixEvidence{items: []domain.Evidence{validEvidence("e1")}}
		t.Setenv("KAS_MTX_JEV_ENDPOINT", fake.URL())
		t.Setenv("KAS_MTX_JEV_API_KEY", "k")
		cfg, err := kas.LoadConfig(strings.NewReader(baseYAML))
		if err != nil {
			t.Fatal(err)
		}
		c, err := kas.New(cfg, kas.Dependencies{EvidenceProvider: ev})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		start := time.Now()
		resp, err := c.Answer(context.Background(), req("Jev 故障问题"))
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed > 600*time.Millisecond {
			t.Fatalf("降级不得超 root deadline（total=400ms）: %s", elapsed)
		}
		return resp
	}

	t.Run("timeout", func(t *testing.T) {
		resp := runWith(t, testutil.JevScript{Body: testutil.ValidJevBody("answer", "g1", 0.9), Delay: 2 * time.Second})
		if !resp.Degraded {
			t.Fatalf("timeout 应降级: %+v", resp)
		}
		if !containsReason(resp.Degradation, kas.ReasonJevUnavailable) && !containsReason(resp.Degradation, kas.ReasonDeadlineExceeded) {
			t.Fatalf("reason 应含 Jev 故障: %v", resp.Degradation)
		}
	})
	t.Run("http_500", func(t *testing.T) {
		resp := runWith(t, testutil.JevScript{Status: 500, Body: `{}`})
		if !resp.Degraded || !containsReason(resp.Degradation, kas.ReasonJevUnavailable) {
			t.Fatalf("5xx 应按序降级: %+v", resp.Degradation)
		}
	})
	t.Run("illegal_action", func(t *testing.T) {
		resp := runWith(t, testutil.JevScript{Body: testutil.ValidJevBody("do_nothing", "g1", 0.9)})
		if !resp.Degraded || !containsReason(resp.Degradation, kas.ReasonJevContractError) {
			t.Fatalf("非法 action 应记 contract error 并降级: %+v", resp.Degradation)
		}
	})
	t.Run("out_of_range_target", func(t *testing.T) {
		resp := runWith(t, testutil.JevScript{Body: testutil.ValidJevBody("answer", "gX", 0.9)})
		if !resp.Degraded || !containsReason(resp.Degradation, kas.ReasonJevContractError) {
			t.Fatalf("越界目标组应拒: %+v", resp.Degradation)
		}
	})
	t.Run("nan_confidence", func(t *testing.T) {
		_ = runWith(t, testutil.JevScript{Body: `{"action":"answer","confidence":NaN}`})
		// JSON 解析层或校验层拒绝即可（细节在 jev 单测）
	})
}

func containsReason(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

// Jev adapter 契约接入（Story 5.2.1.2 DoD：Jev 接入决策契约套件全绿——
// 此处以 fake server 驱动重复核心断言）。
func TestJevAdapterContractViaFake(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{Body: `{"action":"abstain","confidence":0}`})
	defer fake.Close()
	t.Setenv("KAS_MTX_JEV_ENDPOINT", fake.URL())
	t.Setenv("KAS_MTX_JEV_API_KEY", "k")
	jc, err := jev.LoadConfig(cfgJev(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	d, err := jev.New(jc).Decide(context.Background(), kas.DecisionInput{})
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != kas.ActionAbstain {
		t.Fatalf("abstain 决策: %+v", d)
	}
}

func cfgJev() config.JevConfig {
	return config.JevConfig{EndpointEnv: "KAS_MTX_JEV_ENDPOINT", APIKeyEnv: "KAS_MTX_JEV_API_KEY", Model: "m", ModelVersion: "v1", MaxAttempts: 1}
}
