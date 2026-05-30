package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-zoox/config"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestApplyScenarios_liveCacheOverlay(t *testing.T) {
	path := filepath.Join("..", "examples", "scenarios", "design-option-c-list.yaml")
	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: path}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := enrichScenariosFromYAML(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Scenarios.Active = "live"
	if err := ApplyScenarios(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Cache.Host != "redis.internal" {
		t.Fatalf("cache.host = %q, want redis.internal", cfg.Cache.Host)
	}
	if !cfg.Rules[0].Backend.Cache.Enabled {
		t.Fatal("expected live overlay to enable backend.cache")
	}
	if len(cfg.Rules[0].Backend.Cache.Paths) < 2 {
		t.Fatalf("expected cache path rules, got paths=%d", len(cfg.Rules[0].Backend.Cache.Paths))
	}
}

func TestApplyScenarios_insertBeforeWildcardHost(t *testing.T) {
	cfg := Config{
		Scenarios: Scenarios{
			Active: "sh-live",
			Items:  []ScenarioItem{{ID: "sh-live", Label: "上海直播"}},
			overlays: map[string]map[string]any{
				"sh-live": {
					"rules": []any{
						map[string]any{
							"host": "sh.example.com",
							"backend": map[string]any{
								"type": "service",
								"service": map[string]any{
									"name": "sh-origin.internal",
									"port": 8080,
								},
								"cache": map[string]any{
									"enabled": true,
									"ttl":     30,
								},
							},
						},
					},
				},
			},
		},
		Rules: []rule.Rule{
			{
				Host:     "*.example.com",
				HostType: hostTypeWildcard,
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: scenarioTestService("default-origin.internal", 8080),
				},
			},
		},
	}
	if err := ApplyScenarios(&cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("rules len = %d, want 2", len(cfg.Rules))
	}
	if cfg.Rules[0].Host != "sh.example.com" {
		t.Fatalf("inserted rule host = %q, want sh.example.com", cfg.Rules[0].Host)
	}
	if !cfg.Rules[0].Backend.Cache.Enabled {
		t.Fatal("expected overlay cache on inserted sh.example.com rule")
	}
	if cfg.Rules[1].Host != "*.example.com" {
		t.Fatalf("wildcard rule shifted to index 1, got %q", cfg.Rules[1].Host)
	}
	hm, err := MatchHost(cfg.Rules, rule.Backend{}, "sh.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Rule.Host != "sh.example.com" {
		t.Fatalf("MatchHost picked %q, want sh.example.com", hm.Rule.Host)
	}
}

func TestApplyScenarios_appendHostWhenNoCoveringRule(t *testing.T) {
	cfg := Config{
		Scenarios: Scenarios{
			Active: "x",
			Items:  []ScenarioItem{{ID: "x"}},
			overlays: map[string]map[string]any{
				"x": {
					"rules": []any{
						map[string]any{
							"host": "missing.example.com",
							"backend": map[string]any{
								"type": "service",
								"service": map[string]any{
									"name": "extra.internal",
									"port": 8080,
								},
								"cache": map[string]any{"enabled": true},
							},
						},
					},
				},
			},
		},
		Rules: []rule.Rule{{Host: "shop.example.com", Backend: rule.Backend{Type: backendTypeService, Service: scenarioTestService("shop", 8080)}}},
	}
	if err := ApplyScenarios(&cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 2 || cfg.Rules[1].Host != "missing.example.com" {
		t.Fatalf("expected appended rule, got %+v", cfg.Rules)
	}
}

func scenarioTestService(name string, port int64) service.Service {
	return service.Service{Name: name, Port: port}
}

func TestMergeRulesByHostMaps_exactHostMerge(t *testing.T) {
	base := []rule.Rule{{
		Host: "shop.example.com",
		Backend: rule.Backend{
			Type:    backendTypeService,
			Service: scenarioTestService("origin", 8080),
		},
	}}
	patches := []map[string]any{{
		"host": "shop.example.com",
		"backend": map[string]any{
			"cache": map[string]any{"enabled": true, "ttl": 60},
		},
	}}
	out, err := mergeRulesByHostMaps(base, patches)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1 merge not insert", len(out))
	}
	if !out[0].Backend.Cache.Enabled || out[0].Backend.Cache.TTL != 60 {
		t.Fatalf("merge cache: enabled=%v ttl=%d", out[0].Backend.Cache.Enabled, out[0].Backend.Cache.TTL)
	}
	if out[0].Backend.Service.Name != "origin" {
		t.Fatalf("service name = %q, want origin preserved", out[0].Backend.Service.Name)
	}
}

func TestMergeRulesByHostMaps_insertBeforeRegexHost(t *testing.T) {
	base := []rule.Rule{{
		Host:     "^([a-z]+)\\.example\\.com$",
		HostType: hostTypeRegex,
		Backend: rule.Backend{
			Type:    backendTypeService,
			Service: scenarioTestService("regex-origin", 8080),
		},
	}}
	patches := []map[string]any{{
		"host": "sh.example.com",
		"backend": map[string]any{
			"type": "service",
			"service": map[string]any{
				"name": "sh-origin",
				"port": 8080,
			},
		},
	}}
	out, err := mergeRulesByHostMaps(base, patches)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Host != "sh.example.com" {
		t.Fatalf("got rules: %+v", out)
	}
	hm, err := MatchHost(out, rule.Backend{}, "sh.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Rule.Host != "sh.example.com" {
		t.Fatalf("MatchHost = %q", hm.Rule.Host)
	}
	hm2, err := MatchHost(out, rule.Backend{}, "bj.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if hm2.Rule.Host != "^([a-z]+)\\.example\\.com$" {
		t.Fatalf("other host should hit regex rule, got %q", hm2.Rule.Host)
	}
}

func TestMergeRulesByHostMaps_wildcardOtherHostsUnchanged(t *testing.T) {
	base := []rule.Rule{{
		Host:     "*.example.com",
		HostType: hostTypeWildcard,
		Backend: rule.Backend{
			Type:    backendTypeService,
			Service: scenarioTestService("wildcard-origin", 8080),
		},
	}}
	patches := []map[string]any{{
		"host": "sh.example.com",
		"backend": map[string]any{
			"type":    "service",
			"service": map[string]any{"name": "sh-origin", "port": 8080},
		},
	}}
	out, err := mergeRulesByHostMaps(base, patches)
	if err != nil {
		t.Fatal(err)
	}
	hm, err := MatchHost(out, rule.Backend{}, "other.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if hm.Rule.Host != "*.example.com" {
		t.Fatalf("other.example.com should match wildcard, got %q", hm.Rule.Host)
	}
}

func TestRuleHostMatches(t *testing.T) {
	cases := []struct {
		rule rule.Rule
		host string
		want bool
	}{
		{rule: rule.Rule{Host: "a.example.com", HostType: hostTypeExact}, host: "a.example.com", want: true},
		{rule: rule.Rule{Host: "a.example.com", HostType: hostTypeExact}, host: "b.example.com", want: false},
		{rule: rule.Rule{Host: "*.example.com", HostType: hostTypeWildcard}, host: "sh.example.com", want: true},
		{rule: rule.Rule{Host: "^sh\\.example\\.com$", HostType: hostTypeRegex}, host: "sh.example.com", want: true},
		{rule: rule.Rule{Host: "^sh\\.example\\.com$", HostType: hostTypeRegex}, host: "bj.example.com", want: false},
	}
	for _, tc := range cases {
		if got := ruleHostMatches(tc.rule, tc.host); got != tc.want {
			t.Fatalf("ruleHostMatches(%q, %q) = %v, want %v", tc.rule.Host, tc.host, got, tc.want)
		}
	}
}

func TestRuleInsertIndexBeforeMatch(t *testing.T) {
	rules := []rule.Rule{
		{Host: "api.example.com", HostType: hostTypeExact},
		{Host: "*.example.com", HostType: hostTypeWildcard},
	}
	if got := ruleInsertIndexBeforeMatch(rules, "sh.example.com"); got != 1 {
		t.Fatalf("insert index = %d, want 1 (before wildcard)", got)
	}
	if got := ruleInsertIndexBeforeMatch(rules, "api.example.com"); got != 0 {
		t.Fatalf("insert index = %d, want 0 (before exact api rule)", got)
	}
}

func TestFinalizeLoadedConfig_wildcardExactOverlay(t *testing.T) {
	path := filepath.Join("..", "examples", "scenarios", "wildcard-with-exact-overlay.yaml")
	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: path}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := FinalizeLoadedConfig(&cfg, path, raw); err != nil {
		t.Fatal(err)
	}
	// active: default — only baseline wildcard
	if len(cfg.Rules) != 1 || cfg.Rules[0].Host != "*.example.com" {
		t.Fatalf("default active: rules = %+v", cfg.Rules)
	}

	var cfg2 Config
	if err := config.Load(&cfg2, &config.LoadOptions{FilePath: path}); err != nil {
		t.Fatal(err)
	}
	if err := enrichScenariosFromYAML(raw, &cfg2); err != nil {
		t.Fatal(err)
	}
	cfg2.Scenarios.Active = "sh-live"
	if err := ApplyScenarios(&cfg2); err != nil {
		t.Fatal(err)
	}
	if len(cfg2.Rules) != 2 || cfg2.Rules[0].Host != "sh.example.com" {
		t.Fatalf("sh-live: rules = %+v", cfg2.Rules)
	}
	hm, err := MatchHost(cfg2.Rules, rule.Backend{}, "sh.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !hm.Rule.Backend.Cache.Enabled {
		t.Fatal("expected cache on sh.example.com overlay rule")
	}
}

func TestValidateScenariosConfig_duplicateID(t *testing.T) {
	cfg := Config{
		Scenarios: Scenarios{
			Active: "a",
			Items: []ScenarioItem{
				{ID: "a"},
				{ID: "a"},
			},
		},
	}
	if err := ValidateScenariosConfig(&cfg); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestSetScenariosActiveYAML(t *testing.T) {
	in := "scenarios:\n  active: daily\n  items:\n    - id: daily\n    - id: live\n"
	out, err := SetScenariosActiveYAML(in, "live")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "active: live") {
		t.Fatalf("expected active live, got:\n%s", out)
	}
}

func TestResolveActiveScenario_envOverride(t *testing.T) {
	t.Setenv(envScenarioOverride, "live")
	cfg := Config{Scenarios: Scenarios{Active: "daily"}}
	if got := ResolveActiveScenario(&cfg); got != "live" {
		t.Fatalf("got %q", got)
	}
}

func TestListScenarios(t *testing.T) {
	cfg := Config{
		Scenarios: Scenarios{
			Active: "daily",
			Items: []ScenarioItem{
				{ID: "daily", Label: "日常"},
				{ID: "live", Label: "直播"},
			},
		},
	}
	out := ListScenarios(&cfg)
	if out.Active != "daily" || len(out.Scenarios) != 3 {
		t.Fatalf("unexpected list: %+v", out)
	}
	if out.Scenarios[0].ID != DefaultScenarioID || out.Scenarios[0].Active {
		t.Fatalf("expected virtual default first: %+v", out.Scenarios[0])
	}
	if !out.Scenarios[1].Active || out.Scenarios[2].Active {
		t.Fatalf("active flags: %+v", out.Scenarios)
	}
}

func TestApplyScenarios_defaultSkipsOverlay(t *testing.T) {
	path := filepath.Join("..", "examples", "scenarios", "design-option-c-list.yaml")
	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: path}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := enrichScenariosFromYAML(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Scenarios.Active = DefaultScenarioID
	if err := ApplyScenarios(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Cache.Host == "redis.internal" {
		t.Fatal("default scenario should not apply live cache overlay")
	}
}

func TestValidateScenariosConfig_reservedDefaultItemID(t *testing.T) {
	cfg := Config{
		Scenarios: Scenarios{
			Active: DefaultScenarioID,
			Items: []ScenarioItem{
				{ID: DefaultScenarioID},
			},
		},
	}
	if err := ValidateScenariosConfig(&cfg); err == nil {
		t.Fatal("expected reserved id error")
	}
}

func TestFinalizeLoadedConfig_designOptionC(t *testing.T) {
	path := filepath.Join("..", "examples", "scenarios", "design-option-c-list.yaml")
	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: path}); err != nil {
		t.Fatal(err)
	}
	if err := FinalizeLoadedConfig(&cfg, path, nil); err != nil {
		t.Fatal(err)
	}
	if err := ValidateConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Rules[0].Backend.Cache.Enabled {
		t.Fatal("daily scenario should disable cache on shop host")
	}
}
