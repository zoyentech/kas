package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"time"
)

// JevScript 是 fake Jev server 的脚本化响应（Story 5.2.1.1 T2）。
type JevScript struct {
	Status   int           // HTTP 状态码（0 视为 200）
	Body     string        // 响应体（JSON）
	Delay    time.Duration // 响应前延迟（模拟慢响应）
	Calls    int32         // 调用计数（测试断言单次尝试）
	LastAuth string        // 捕获的 Authorization 头（扫描测试用）
	LastBody string        // 捕获的请求体
	// BalanceBody 非空时 GET /balance 返回该 JSON（如 {"amount":42,"unit":"credit"}）。
	BalanceBody string
	// BalanceStatus /balance 状态码（0 视为 200；置 5xx 可模拟余额查询故障）。
	BalanceStatus int
}

// JevFakeServer 是脚本化 Jev 端点（覆盖 200/timeout/5xx/畸形体）。
type JevFakeServer struct {
	mu     sync.Mutex
	script JevScript
	srv    *httptest.Server
}

// NewJevFakeServer 启动 fake server。
func NewJevFakeServer(script JevScript) *JevFakeServer {
	f := &JevFakeServer{script: script}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/balance" {
			f.mu.Lock()
			status, body := f.script.BalanceStatus, f.script.BalanceBody
			f.mu.Unlock()
			if status == 0 {
				status = http.StatusOK
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
			return
		}
		f.mu.Lock()
		f.script.Calls++
		f.script.LastAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		f.script.LastBody = string(buf)
		script := f.script
		f.mu.Unlock()
		if script.Delay > 0 {
			select {
			case <-time.After(script.Delay):
			case <-r.Context().Done():
				return
			}
		}
		status := script.Status
		if status == 0 {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(script.Body))
	}))
	return f
}

// URL 返回端点地址。
func (f *JevFakeServer) URL() string { return f.srv.URL }

// Close 关闭 server。
func (f *JevFakeServer) Close() { f.srv.Close() }

// Calls 返回调用计数（单次尝试断言）。
func (f *JevFakeServer) Calls() int32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.script.Calls
}

// LastAuth 返回最近一次 Authorization 头。
func (f *JevFakeServer) LastAuth() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.script.LastAuth
}

// LastRequestBody 返回最近一次请求体。
func (f *JevFakeServer) LastRequestBody() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.script.LastBody
}

// ValidJevBody 返回合法 Jev 响应体（目标组 g1、置信 0.9、版本捕获字段）。
func ValidJevBody(action, groupID string, confidence float64) string {
	b, _ := json.Marshal(map[string]interface{}{
		"action":           action,
		"target_group_ids": []string{groupID},
		"need_supplement":  false,
		"reason_code":      "ok",
		"confidence":       confidence,
		"model":            "jev-test",
		"model_version":    "v9",
		"answer_text":      "回答正文",
	})
	return string(b)
}

// SetBalanceBody 动态更新 /balance 响应（状态迁移演示/测试）。
func (f *JevFakeServer) SetBalanceBody(body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.script.BalanceBody = body
}

// SetBalanceAmount 以 credit 单位生成余额 JSON 并更新脚本（快捷方式）。
func (f *JevFakeServer) SetBalanceAmount(amount float64) {
	f.SetBalanceBody(`{"amount":` + strconv.FormatFloat(amount, 'f', 2, 64) + `,"unit":"credit"}`)
}
