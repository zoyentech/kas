package config

import (
	"strings"
	"testing"
)

// TestVersionOrderIndependent 同一配置不同 YAML 字段顺序 → 同一 config_version
// （Story 2.2.1.1 T1 顺序无关性）。
func TestVersionOrderIndependent(t *testing.T) {
	a := validConfig()
	// 手工重排：default_max_age map 与 include 列表换序构造第二份
	b := validConfig()
	b.Evidence.Validity.DefaultMaxAge = map[string]Duration{
		"service_notice": b.Evidence.Validity.DefaultMaxAge["service_notice"],
		"product_manual": b.Evidence.Validity.DefaultMaxAge["product_manual"],
	}
	rev := make([]string, 0, len(b.Evidence.Cache.Include))
	for i := len(b.Evidence.Cache.Include) - 1; i >= 0; i-- {
		rev = append(rev, b.Evidence.Cache.Include[i])
	}
	b.Evidence.Cache.Include = rev
	revTr := make([]string, 0, len(b.Tracing.Include))
	for i := len(b.Tracing.Include) - 1; i >= 0; i-- {
		revTr = append(revTr, b.Tracing.Include[i])
	}
	b.Tracing.Include = revTr

	va, errA := Version(a)
	vb, errB := Version(b)
	if errA != nil || errB != nil {
		t.Fatalf("版本生成失败: %v / %v", errA, errB)
	}
	if va != vb {
		t.Fatalf("字段顺序无关：两份同语义配置版本应相等，实际 %s != %s", va, vb)
	}
}

// TestVersionDeterministic 幂等：同配置多次生成一致。
func TestVersionDeterministic(t *testing.T) {
	cfg := validConfig()
	v1 := MustVersion(cfg)
	for i := 0; i < 10; i++ {
		if got := MustVersion(cfg); got != v1 {
			t.Fatalf("版本不幂等: %s != %s", got, v1)
		}
	}
	if len(v1) != VersionLength {
		t.Fatalf("版本长度应为 %d hex，实际 %d", VersionLength, len(v1))
	}
}

// TestVersionChangesOnMaterialChange 任何影响行为的字段变更都改变版本。
func TestVersionChangesOnMaterialChange(t *testing.T) {
	base := MustVersion(validConfig())
	mutations := []struct {
		name string
		f    func(*Config)
	}{
		{"total", func(c *Config) { c.Timeouts.Total = D(4800_000_001); c.Timeouts.Reserve = D(600_000_001) }},
		{"top_k", func(c *Config) { c.Evidence.TopK = 10 }},
		{"cache_ttl", func(c *Config) { c.Evidence.Cache.TTL = D(121_000_000_000) }},
		{"jev_model", func(c *Config) { c.Decision.Jev.Model = "jev-v2" }},
		{"max_citations", func(c *Config) { c.Answer.MaxCitations = 4 }},
		{"max_age_policy", func(c *Config) {
			c.Evidence.Validity.DefaultMaxAge["product_manual"] = D(719 * 3600e9)
		}},
	}
	for _, m := range mutations {
		cfg := validConfig()
		m.f(&cfg)
		if got := MustVersion(cfg); got == base {
			t.Errorf("%s 变更后版本未变化（%s）", m.name, got)
		}
	}
}

// TestVersionSecretSemantics secret 语义（Story 2.2.1.1 T2）：
//   - env 变量值变化（本就不进入 Config）→ 版本不变
//   - env 引用名变化 → 版本变化
func TestVersionSecretSemantics(t *testing.T) {
	t.Setenv("KAS_JEV_API_KEY", "secret-value-A")
	cfg := validConfig()
	v1 := MustVersion(cfg)
	t.Setenv("KAS_JEV_API_KEY", "secret-value-B")
	if got := MustVersion(cfg); got != v1 {
		t.Fatal("secret 值不参与版本计算，版本不应变化")
	}
	cfg2 := validConfig()
	cfg2.Decision.Jev.APIKeyEnv = "KAS_JEV_API_KEY_V2"
	if got := MustVersion(cfg2); got == v1 {
		t.Fatal("secret 引用名变化是配置变化，版本应变化")
	}
}

// TestVersionStableAcrossProcesses 类跨进程稳定性：JSON map 键序确定
// （default_max_age 以不同插入序构造 → 同版本）。
func TestVersionStableAcrossProcesses(t *testing.T) {
	mk := func(order ...string) Config {
		cfg := validConfig()
		m := map[string]Duration{}
		for _, k := range order {
			m[k] = cfg.Evidence.Validity.DefaultMaxAge[k]
		}
		cfg.Evidence.Validity.DefaultMaxAge = m
		return cfg
	}
	a := MustVersion(mk("product_manual", "service_notice"))
	b := MustVersion(mk("service_notice", "product_manual"))
	if a != b {
		t.Fatalf("map 插入序不应影响版本: %s != %s", a, b)
	}
}

// TestCanonicalDoesNotMutateInput Canonical 排序集合不得修改入参切片。
func TestCanonicalDoesNotMutateInput(t *testing.T) {
	cfg := validConfig()
	orig := append([]string(nil), cfg.Evidence.Cache.Include...)
	// 构造逆序输入
	rev := make([]string, len(orig))
	for i := range orig {
		rev[i] = orig[len(orig)-1-i]
	}
	cfg.Evidence.Cache.Include = rev
	if _, err := Canonical(cfg); err != nil {
		t.Fatal(err)
	}
	if strings.Join(cfg.Evidence.Cache.Include, ",") != strings.Join(rev, ",") {
		t.Fatal("Canonical 修改了入参切片")
	}
}
