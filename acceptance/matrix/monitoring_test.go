package matrix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// monitoringRoot 监控栈目录（相对本包）。
const monitoringRoot = "../../monitoring"

// requiredPanels 看板必须覆盖的面板标题集（Story 8.7.1.2 验收）。
var requiredPanels = []string{
	"问答成功率 %", "QPS", "请求时延分位", "阶段耗时 p95", "阶段超时 /s",
	"Evidence 缓存命中率 %", "降级路由 /s", "Provider 错误 /s", "余额", "证据过滤 /s",
	"Jev 熔断", "Evidence Provider", "Jev", "组件余额总览",
}

// prefixPanels 以前缀匹配的标题（repeat 模板面板标题含变量后缀）。
var prefixPanels = []string{"余额状态 · "}

// TestMonitoringStackFiles 栈文件齐备与关键内容。
func TestMonitoringStackFiles(t *testing.T) {
	for _, f := range []string{
		"docker-compose.yml", "prometheus/prometheus.yml",
		"grafana/provisioning/datasources/prometheus.yml",
		"grafana/provisioning/dashboards/provider.yml",
		"grafana/dashboards/kas-overview.json",
	} {
		if _, err := os.Stat(filepath.Join(monitoringRoot, f)); err != nil {
			t.Errorf("监控栈缺文件 %s: %v", f, err)
		}
	}
	compose, err := os.ReadFile(filepath.Join(monitoringRoot, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"prom/prometheus:v", "grafana/grafana:", "host.docker.internal"} {
		if !strings.Contains(string(compose), want) {
			t.Errorf("compose 缺关键配置 %q", want)
		}
	}
	// 版本固定（禁止 latest 漂移）
	if strings.Contains(string(compose), ":latest") {
		t.Error("镜像禁止使用 latest")
	}
	prom, err := os.ReadFile(filepath.Join(monitoringRoot, "prometheus/prometheus.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"job_name: kas", "/metrics", "scrape_interval"} {
		if !strings.Contains(string(prom), want) {
			t.Errorf("prometheus.yml 缺 %q", want)
		}
	}
}

// TestDashboardStructure 看板 JSON 结构校验：面板齐备、表达式非空、
// 数据源引用一致、状态 gauge 口径存在。
func TestDashboardStructure(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(monitoringRoot, "grafana/dashboards/kas-overview.json"))
	if err != nil {
		t.Fatal(err)
	}
	var dash struct {
		Title  string `json:"title"`
		UID    string `json:"uid"`
		Panels []struct {
			Title   string `json:"title"`
			Targets []struct {
				Expr string `json:"expr"`
			} `json:"targets"`
			Datasource struct {
				UID string `json:"uid"`
			} `json:"datasource"`
		} `json:"panels"`
	}
	if err := json.Unmarshal(raw, &dash); err != nil {
		t.Fatalf("看板 JSON 非法: %v", err)
	}
	if dash.UID != "kas-overview" || len(dash.Panels) == 0 {
		t.Fatalf("看板基础字段异常: uid=%s panels=%d", dash.UID, len(dash.Panels))
	}
	titles := map[string]bool{}
	var allExprs []string
	for _, p := range dash.Panels {
		titles[p.Title] = true
		if p.Datasource.UID != "kas-prom" {
			t.Errorf("面板 %q 数据源引用异常: %q", p.Title, p.Datasource.UID)
		}
		for _, tg := range p.Targets {
			if strings.TrimSpace(tg.Expr) == "" {
				t.Errorf("面板 %q 存在空表达式", p.Title)
			}
			allExprs = append(allExprs, tg.Expr)
		}
	}
	for _, want := range requiredPanels {
		if !titles[want] {
			t.Errorf("看板缺面板 %q", want)
		}
	}
	for _, prefix := range prefixPanels {
		found := false
		for title := range titles {
			if strings.HasPrefix(title, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("看板缺前缀面板 %q（按源 repeat）", prefix)
		}
	}
	// 多源模板变量存在
	var tpl struct {
		Templating struct {
			List []struct {
				Name string `json:"name"`
			} `json:"list"`
		} `json:"templating"`
	}
	if err := json.Unmarshal(raw, &tpl); err != nil {
		t.Fatal(err)
	}
	hasVar := false
	for _, v := range tpl.Templating.List {
		if v.Name == "balance_source" {
			hasVar = true
		}
	}
	if !hasVar {
		t.Error("看板缺 balance_source 模板变量（多源分列）")
	}
	// 关键口径抽查：成功率/分位/余额/熔断表达式必须出现
	joined := strings.Join(allExprs, "\n")
	for _, want := range []string{
		"kas_requests_total", "histogram_quantile", "kas_balance_state",
		"kas_provider_breaker_state", "kas_evidence_cache_total", "kas_decision_fallback_total",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("看板表达式缺指标 %s", want)
		}
	}
}
