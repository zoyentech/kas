package adapters_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zoyentech/kas/internal/balance"
	"github.com/zoyentech/kas/internal/balance/adapters"
	"github.com/zoyentech/kas/internal/domain"
	"github.com/zoyentech/kas/internal/testutil"
)

func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }

// TestDeepSeekContract DeepSeek 余额契约（参照 ai-support-platform #1243 实现）。
func TestDeepSeekContract(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[` +
			`{"currency":"CNY","total_balance":"110.50","granted_balance":"10.00","topped_up_balance":"100.50"}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("KAS_TEST_DS_TOKEN", "sk-test")
	t.Setenv("KAS_TEST_DS_EP", srv.URL+"/user/balance")
	src, err := adapters.NewDeepSeek("KAS_TEST_DS_TOKEN", "KAS_TEST_DS_EP")
	if err != nil {
		t.Fatal(err)
	}
	info, err := src.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Source != "deepseek" || info.Amount != 110.5 || info.Unit != "CNY" {
		t.Fatalf("info 异常: %+v", info)
	}

	t.Run("multi_currency_prefers_cny", func(t *testing.T) {
		// 多币种：CNY 排后也应被选中
		mux2 := http.NewServeMux()
		mux2.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[` +
				`{"currency":"USD","total_balance":"99"},{"currency":"CNY","total_balance":"7.25"}]}`))
		})
		srv2 := httptest.NewServer(mux2)
		defer srv2.Close()
		t.Setenv("KAS_TEST_DS2", "k")
		t.Setenv("KAS_TEST_DS2_EP", srv2.URL+"/b")
		src2, _ := adapters.NewDeepSeek("KAS_TEST_DS2", "KAS_TEST_DS2_EP")
		info2, err := src2.Balance(context.Background())
		if err != nil || info2.Amount != 7.25 || info2.Unit != "CNY" {
			t.Fatalf("多币种应选 CNY: %+v %v", info2, err)
		}
	})

	t.Run("not_available", func(t *testing.T) {
		mux3 := http.NewServeMux()
		mux3.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"is_available":false,"balance_infos":[]}`))
		})
		srv3 := httptest.NewServer(mux3)
		defer srv3.Close()
		t.Setenv("KAS_TEST_DS3", "k")
		t.Setenv("KAS_TEST_DS3_EP", srv3.URL+"/b")
		src3, _ := adapters.NewDeepSeek("KAS_TEST_DS3", "KAS_TEST_DS3_EP")
		if _, err := src3.Balance(context.Background()); err == nil {
			t.Fatal("is_available=false 应报错（→unknown 态）")
		}
	})

	t.Run("malformed", func(t *testing.T) {
		mux4 := http.NewServeMux()
		mux4.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{oops`))
		})
		srv4 := httptest.NewServer(mux4)
		defer srv4.Close()
		t.Setenv("KAS_TEST_DS4", "k")
		t.Setenv("KAS_TEST_DS4_EP", srv4.URL+"/b")
		src4, _ := adapters.NewDeepSeek("KAS_TEST_DS4", "KAS_TEST_DS4_EP")
		if _, err := src4.Balance(context.Background()); !errors.Is(err, domain.ErrProviderContract) {
			t.Fatalf("畸形体应契约错误: %v", err)
		}
	})

	t.Run("env_missing", func(t *testing.T) {
		if _, err := adapters.NewDeepSeek("KAS_TEST_DS_NOPE", ""); !errors.Is(err, domain.ErrInvalidConfig) {
			t.Fatalf("env 缺失应 ErrInvalidConfig: %v", err)
		}
	})
}

// TestAliyunBSSContract BSS QueryAccountBalance：签名请求可达 + 成功/失败语义。
// 通过本地 fake 校验请求形态（含 Signature/AccessKeyId/Action）与响应解析。
func TestAliyunBSSContract(t *testing.T) {
	var gotQuery strings.Builder
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		gotQuery.Reset()
		gotQuery.WriteString(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Success":true,"Data":{"AvailableCashAmount":"88.66","Currency":"CNY"}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("KAS_TEST_AK", "test-ak")
	t.Setenv("KAS_TEST_SK", "test-sk")
	t.Setenv("KAS_TEST_BSS_EP", srv.URL)
	src, err := adapters.NewAliyunBSS("KAS_TEST_AK", "KAS_TEST_SK", "", "KAS_TEST_BSS_EP")
	if err != nil {
		t.Fatal(err)
	}
	info, err := src.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Source != "aliyun" || info.Amount != 88.66 || info.Unit != "CNY" {
		t.Fatalf("info 异常: %+v", info)
	}
	// 请求形态：POP RPC 关键参数齐全
	q := gotQuery.String()
	for _, want := range []string{"Action=QueryAccountBalance", "AccessKeyId=test-ak", "SignatureMethod=HMAC-SHA1", "Signature="} {
		if !strings.Contains(q, want) {
			t.Errorf("BSS 请求缺 %q: %s", want, q)
		}
	}

	t.Run("failure_semantics", func(t *testing.T) {
		mux2 := http.NewServeMux()
		mux2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"Success":false}`))
		})
		srv2 := httptest.NewServer(mux2)
		defer srv2.Close()
		t.Setenv("KAS_TEST_SK2", "k2")
		t.Setenv("KAS_TEST_BSS_EP2", srv2.URL)
		src2, _ := adapters.NewAliyunBSS("KAS_TEST_AK", "KAS_TEST_SK2", "", "KAS_TEST_BSS_EP2")
		if _, err := src2.Balance(context.Background()); err == nil {
			t.Fatal("Success=false 应报错")
		}
	})

	t.Run("env_missing", func(t *testing.T) {
		if _, err := adapters.NewAliyunBSS("KAS_TEST_AK_NOPE", "KAS_TEST_SK_NOPE", "", ""); !errors.Is(err, domain.ErrInvalidConfig) {
			t.Fatalf("AK/SK 缺失应 ErrInvalidConfig: %v", err)
		}
	})
}

var _ = balance.StateNormal
