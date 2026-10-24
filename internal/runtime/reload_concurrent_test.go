package runtime_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/zoyentech/kas/internal/runtime"
)

// TestReloadConcurrentNoVersionMixing F-011 并发语义（Story 2.2.2.2 T3）：
// 在途"请求"（持有一次 Current() 的快照）在中途 Reload 后仍读旧版本；
// 新请求读新版本；二者绝无混合。千次迭代 + -race。
func TestReloadConcurrentNoVersionMixing(t *testing.T) {
	m := runtime.NewManager(mustBuild(t, validYAML))
	oldV := m.Version()
	newV := mustBuild(t, mutatedYAML).Version()

	const iterations = 1000
	var wg sync.WaitGroup
	errs := make(chan error, iterations*3)

	// 在途请求：读取一次快照后经过一次 Reload 周期，全程使用旧版本。
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snap := m.Current() // 请求开始读取一次并全程持有
			inFlightV := snap.Version()
			// 模拟长请求期间发生 Reload（等待新版本出现）
			for j := 0; j < 2000 && m.Version() == inFlightV; j++ {
				continue
			}
			// 断言：持有的快照版本不受后续替换影响，且属于旧或新二者之一
			if snap.Version() != inFlightV {
				errs <- fmt.Errorf("在途快照版本被篡改: %s -> %s", inFlightV, snap.Version())
			}
			if inFlightV != oldV && inFlightV != newV {
				errs <- fmt.Errorf("在途版本 %s 既非旧 %s 也非新 %s", inFlightV, oldV, newV)
			}
		}()
	}
	// Reload 方：新旧交替
	for i := 0; i < iterations/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := runtime.Reload(context.Background(), m, strings.NewReader(validYAML)); err != nil {
				errs <- err
			}
			if _, err := runtime.Reload(context.Background(), m, strings.NewReader(mutatedYAML)); err != nil {
				errs <- err
			}
		}()
	}
	// 无效 Reload 并发混入：全部失败且不影响版本空间
	for i := 0; i < iterations/4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bad := strings.Replace(validYAML, "total: 4800ms", "total: 99ms", 1)
			if _, err := runtime.Reload(context.Background(), m, strings.NewReader(bad)); err == nil {
				errs <- errors.New("无效 Reload 意外成功")
			}
		}()
	}
	// 新请求：读到的版本只可能是旧或新
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := m.Version()
			if v != oldV && v != newV {
				errs <- fmt.Errorf("新请求读到第三种版本 %s", v)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestConcurrentReplaceLoadStress 替换瞬间并发 Load 只见旧或新，无混合对象。
func TestConcurrentReplaceStress(t *testing.T) {
	snapA := mustBuild(t, validYAML)
	snapB := mustBuild(t, mutatedYAML)
	m := runtime.NewManager(snapA)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					s := m.Current()
					if s != nil && s.Version() != snapA.Version() && s.Version() != snapB.Version() {
						t.Errorf("Load 读到未知版本 %s", s.Version())
						return
					}
				}
			}
		}()
	}
	for i := 0; i < 2000; i++ {
		if i%2 == 0 {
			m.Replace(snapA)
		} else {
			m.Replace(snapB)
		}
	}
	close(stop)
	wg.Wait()
}
