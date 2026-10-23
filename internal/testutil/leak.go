package testutil

import (
	"testing"

	"go.uber.org/goleak"
)

// VerifyTestMain 在 TestMain 中挂接 goleak，测试结束后检测 goroutine 泄漏
// （D-9：防 context/后台任务泄漏）。用法：
//
//	func TestMain(m *testing.M) { testutil.VerifyTestMain(m) }
//
// 需要豁免已知后台 goroutine 的包，改用 goleak.VerifyTestMain 并配置
// goleak.IgnoreAnyFunction("…")。
func VerifyTestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// CheckLeaks 在单个测试内即时检测泄漏（用于并发场景测试的定点断言）。
func CheckLeaks(t testing.TB) {
	t.Helper()
	if err := goleak.Find(goleak.IgnoreCurrent()); err != nil {
		t.Fatalf("检测到 goroutine 泄漏: %v", err)
	}
}
