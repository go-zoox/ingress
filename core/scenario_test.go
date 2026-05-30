package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-zoox/config"
	"github.com/go-zoox/ingress/core/rule"
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

func TestApplyScenarios_unknownHost(t *testing.T) {
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
								"cache": map[string]any{"enabled": true},
							},
						},
					},
				},
			},
		},
		Rules: []rule.Rule{{Host: "shop.example.com"}},
	}
	if err := ApplyScenarios(&cfg); err == nil {
		t.Fatal("expected unknown host error")
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
