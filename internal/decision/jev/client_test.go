package jev_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kas-runtime/kas/internal/config"
	"github.com/kas-runtime/kas/internal/decision/jev"
	"github.com/kas-runtime/kas/internal/domain"
	"github.com/kas-runtime/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

func jevCfg(endpointEnv string) config.JevConfig {
	return config.JevConfig{
		EndpointEnv: endpointEnv, APIKeyEnv: "KAS_TEST_API_KEY",
		Model: "jev-latest", ModelVersion: "stable", MaxAttempts: 1,
	}
}

func decisionInput() domain.DecisionInput {
	return domain.DecisionInput{
		Question: "服务支持哪些配置方式？",
		Groups: []domain.CandidateGroup{{
			ID: "g1", Score: 0.9,
			Evidence: []domain.EvidenceRef{{EvidenceID: "doc-10#seg-2", SourceID: "doc-10", ContentVersion: "v3"}},
		}},
		ConfigVersion: "v1",
	}
}

// TestLoadConfigEnvValidation env 注入：缺失报错且错误只含 env 名不含值
// （Story 5.2.1.1 T1）。
func TestLoadConfigEnvValidation(t *testing.T) {
	t.Setenv("KAS_TEST_ENDPOINT", "http://jev.example")
	t.Setenv("KAS_TEST_API_KEY", "secret-key-value-123")
	cfg, err := jev.LoadConfig(jevCfg("KAS_TEST_ENDPOINT"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	// Stringer redaction：api_key 值零出现
	if strings.Contains(cfg.String(), "secret-key-value-123") {
		t.Fatal("String 泄漏 api_key")
	}

	// 缺失 endpoint
	_, err = jev.LoadConfig(jevCfg("KAS_MISSING_ENDPOINT"), time.Second)
	if err == nil || !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("缺失 endpoint 应 ErrInvalidConfig: %v", err)
	}
	if !strings.Contains(err.Error(), "KAS_MISSING_ENDPOINT") {
		t.Fatalf("错误应含 env 名: %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("错误不得含任何 api_key 值")
	}
	// 缺失 api key
	t.Setenv("KAS_TEST_ENDPOINT2", "http://jev.example")
	_, err = jev.LoadConfig(config.JevConfig{EndpointEnv: "KAS_TEST_ENDPOINT2", APIKeyEnv: "KAS_MISSING_KEY"}, time.Second)
	if err == nil || !strings.Contains(err.Error(), "KAS_MISSING_KEY") {
		t.Fatalf("缺失 api key 应报 env 名: %v", err)
	}
}

// TestJevHappyPath fake server 200：请求映射、决策返回、版本捕获
// （Story 5.2.1.1 T2 / 5.2.1.2）。
func TestJevHappyPath(t *testing.T) {
	fake := testutil.NewJevFakeServer(testutil.JevScript{
		Body: testutil.ValidJevBody("answer", "g1", 0.9),
	})
	defer fake.Close()
	t.Setenv("KAS_TEST_ENDPOINT", fake.URL())
	t.Setenv("KAS_TEST_API_KEY", "the-key")
	cfg, err := jev.LoadConfig(jevCfg("KAS_TEST_ENDPOINT"), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c := jev.New(cfg)
	d, err := c.Decide(context.Background(), decisionInput())
	if err != nil {
		t.Fatal(err)
	}
	if d.Action != domain.ActionAnswer || d.Provider != "jev" || d.Confidence != 0.9 {
		t.Fatalf("决策异常: %+v", d)
	}
	// 版本捕获：响应优先
	if d.Model != "jev-test" || d.ModelVersion != "v9" {
		t.Fatalf("版本捕获异常: %s %s", d.Model, d.ModelVersion)
	}
	if d.AnswerText != "回答正文" {
		t.Fatalf("回答草稿异常: %q", d.AnswerText)
	}
	// 请求映射：候选组必要字段 + Bearer 认证
	body := fake.LastRequestBody()
	for _, want := range []string{`"candidate_groups"`, `"g1"`, `"doc-10#seg-2"`, `"config_version":"v1"`} {
		if !strings.Contains(body, want) {
			t.Errorf("请求体缺 %s: %s", want, body)
		}
	}
	if fake.LastAuth() != "Bearer the-key" {
		t.Fatalf("Authorization 异常: %q", fake.LastAuth())
	}
	// 单次尝试
	if fake.Calls() != 1 {
		t.Fatalf("调用次数 = %d（单次尝试）", fake.Calls())
	}
}

// TestJevFaultScenarios fake server 故障矩阵（docs/05 §2 Jev 行）。
func TestJevFaultScenarios(t *testing.T) {
	makeClient := func(t *testing.T, script testutil.JevScript) (*jev.Client, *testutil.JevFakeServer) {
		fake := testutil.NewJevFakeServer(script)
		t.Setenv("KAS_TEST_ENDPOINT", fake.URL())
		t.Setenv("KAS_TEST_API_KEY", "k")
		cfg, err := jev.LoadConfig(jevCfg("KAS_TEST_ENDPOINT"), 200*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		return jev.New(cfg), fake
	}
	t.Run("http_500", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Status: http.StatusInternalServerError, Body: `{}`})
		defer fake.Close()
		if _, err := c.Decide(context.Background(), decisionInput()); err == nil {
			t.Fatal("5xx 应报错（链降级）")
		}
	})
	t.Run("timeout", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: testutil.ValidJevBody("answer", "g1", 0.9), Delay: 2 * time.Second})
		defer fake.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		if _, err := c.Decide(ctx, decisionInput()); err == nil {
			t.Fatal("慢响应应超时")
		}
		if time.Since(start) > 500*time.Millisecond {
			t.Fatal("超时应受 transport/ctx 双重约束")
		}
	})
	t.Run("malformed_body", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: `{not json`})
		defer fake.Close()
		_, err := c.Decide(context.Background(), decisionInput())
		if !errors.Is(err, domain.ErrProviderContract) {
			t.Fatalf("畸形体应分类 contract error: %v", err)
		}
	})
	t.Run("unknown_field", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: `{"action":"answer","surprise":1}`})
		defer fake.Close()
		_, err := c.Decide(context.Background(), decisionInput())
		if !errors.Is(err, domain.ErrProviderContract) {
			t.Fatalf("未知字段应拒绝: %v", err)
		}
	})
	t.Run("illegal_action", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: testutil.ValidJevBody("do_nothing", "g1", 0.9)})
		defer fake.Close()
		_, err := c.Decide(context.Background(), decisionInput())
		if !errors.Is(err, domain.ErrProviderContract) || !strings.Contains(err.Error(), "action") {
			t.Fatalf("非法 action 应拒绝: %v", err)
		}
	})
	t.Run("unknown_target_group", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: testutil.ValidJevBody("answer", "gX", 0.9)})
		defer fake.Close()
		_, err := c.Decide(context.Background(), decisionInput())
		if !errors.Is(err, domain.ErrProviderContract) {
			t.Fatalf("越界目标组应拒绝: %v", err)
		}
	})
	t.Run("nan_confidence", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: `{"action":"answer","confidence":NaN}`})
		defer fake.Close()
		if _, err := c.Decide(context.Background(), decisionInput()); err == nil {
			t.Fatal("NaN 应拒绝（JSON 解析层或校验层）")
		}
	})
	t.Run("no_second_call_on_timeout", func(t *testing.T) {
		c, fake := makeClient(t, testutil.JevScript{Body: testutil.ValidJevBody("answer", "g1", 0.9), Delay: 500 * time.Millisecond})
		defer fake.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, _ = c.Decide(ctx, decisionInput())
		if fake.Calls() != 1 {
			t.Fatalf("单次尝试语义被违反: %d", fake.Calls())
		}
	})
}
