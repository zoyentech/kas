package balance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoyentech/kas/internal/balance"
	"github.com/zoyentech/kas/internal/config"
	"github.com/zoyentech/kas/internal/obs"
)

func bcfgMulti(interval time.Duration) config.BalanceConfig {
	low, critical := 100.0, 20.0
	return config.BalanceConfig{
		Enabled: true, Interval: config.D(interval),
		LowThreshold: low, CriticalThreshold: critical,
	}
}

// TestManagerMultiSource 多源注册/独立阈值/快照按源（Story：F-016 多源）。
func TestManagerMultiSource(t *testing.T) {
	bus := obs.NewBus()
	m := balance.NewManager(bus)
	defer m.Close()

	m.Upsert("jev", &fakeSource{info: balance.Info{Source: "jev", Amount: 500, Unit: "credit"}}, bcfgMulti(time.Minute))
	// deepseek 用独立阈值：low=1000（500 即 low）
	ov := 1000.0
	cfgDS := bcfgMulti(time.Minute)
	cfgDS.Sources = map[string]config.BalanceSourceConfig{"deepseek": {LowThreshold: &ov}}
	m.Upsert("deepseek", &fakeSource{info: balance.Info{Source: "deepseek", Amount: 500, Unit: "CNY"}}, cfgDS)

	names := m.Names()
	if len(names) != 2 || names[0] != "deepseek" || names[1] != "jev" {
		t.Fatalf("源清单异常: %v", names)
	}
	// 同步驱动各源一轮
	m.PollForTest(t, "jev")
	m.PollForTest(t, "deepseek")

	if _, st, _, ok := m.Snapshot("jev"); !ok || st != balance.StateNormal {
		t.Fatalf("jev 应 normal（500>100）: %s %v", st, ok)
	}
	if _, st, _, ok := m.Snapshot("deepseek"); !ok || st != balance.StateLow {
		t.Fatalf("deepseek 应 low（500≤1000 且 >20，源级覆盖生效）: %s %v", st, ok)
	}
	// DefaultName 优先 jev
	if name, ok := m.DefaultName(); !ok || name != "jev" {
		t.Fatalf("默认源应 jev: %s", name)
	}
}

// TestManagerHotUpsertReplace 热替换源：新源即刻生效、旧轮询停止（goleak 由 TestMain 保证）。
func TestManagerHotUpsertReplace(t *testing.T) {
	m := balance.NewManager(nil)
	defer m.Close()
	m.Upsert("deepseek", &fakeSource{err: errors.New("down")}, bcfgMulti(10*time.Millisecond))
	m.PollForTest(t, "deepseek")
	if _, st, _, _ := m.Snapshot("deepseek"); st != balance.StateUnknown {
		t.Fatalf("旧源故障应 unknown: %s", st)
	}
	// 热替换：换新凭据/端点的源实例（50 ∈ (20,100] → low，恢复成功态）
	m.Upsert("deepseek", &fakeSource{info: balance.Info{Source: "deepseek", Amount: 50, Unit: "CNY"}}, bcfgMulti(time.Minute))
	m.PollForTest(t, "deepseek")
	info, st, _, _ := m.Snapshot("deepseek")
	if st != balance.StateLow || info.Amount != 50 {
		t.Fatalf("新源应生效且 low（50∈(20,100]）: %+v %s", info, st)
	}
}

// TestManagerSetConfig 热更新阈值（不换源）。
func TestManagerSetConfig(t *testing.T) {
	m := balance.NewManager(nil)
	defer m.Close()
	m.Upsert("aliyun", &fakeSource{info: balance.Info{Source: "aliyun", Amount: 15, Unit: "CNY"}}, bcfgMulti(time.Minute))
	m.PollForTest(t, "aliyun")
	if _, st, _, _ := m.Snapshot("aliyun"); st != balance.StateCritical {
		t.Fatalf("15≤20 应 critical: %s", st)
	}
	// 阈值热调低：critical=5 → 15 ∈ (5,100] 变 low
	critical := 5.0
	cfg := bcfgMulti(time.Minute)
	cfg.CriticalThreshold = critical
	cfg.Sources = map[string]config.BalanceSourceConfig{}
	if !m.SetConfig("aliyun", cfg) {
		t.Fatal("SetConfig 应成功")
	}
	m.PollForTest(t, "aliyun")
	if _, st, _, _ := m.Snapshot("aliyun"); st != balance.StateLow {
		t.Fatalf("阈值热更新后 15∈(5,100] 应 low: %s", st)
	}
}

// TestManagerRemoveAndEmpty 移除源与空默认。
func TestManagerRemoveAndEmpty(t *testing.T) {
	m := balance.NewManager(nil)
	m.Upsert("host", &fakeSource{info: balance.Info{Source: "host", Amount: 1, Unit: "u"}}, bcfgMulti(time.Minute))
	if _, ok := m.DefaultName(); !ok {
		t.Fatal("应存在默认源")
	}
	m.Remove("host")
	if _, ok := m.DefaultName(); ok {
		t.Fatal("移除后应无默认源")
	}
	if _, _, _, ok := m.Snapshot("host"); ok {
		t.Fatal("移除后快照应 miss")
	}
}

// snapSource 供 Health 快照对账。
func TestManagerSnapshotAll(t *testing.T) {
	m := balance.NewManager(nil)
	defer m.Close()
	m.Upsert("jev", &fakeSource{info: balance.Info{Source: "jev", Amount: 500, Unit: "credit"}}, bcfgMulti(time.Minute))
	m.PollForTest(t, "jev")
	all := m.SnapshotAll()
	if len(all) != 1 || all["jev"].State != balance.StateNormal {
		t.Fatalf("SnapshotAll 异常: %+v", all)
	}
}

var _ = context.Background
