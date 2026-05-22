package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitAndMergeConfigModules(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "admin-console", "ingress.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample ingress not found")
	}
	content := string(data)
	modules, err := SplitConfigModules(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) < 7 {
		t.Fatalf("expected modules, got %d", len(modules))
	}

	cur := ""
	for _, mod := range modules {
		if strings.TrimSpace(mod.YAML) == "" {
			continue
		}
		cur, err = MergeConfigModule(cur, mod.ID, mod.YAML)
		if err != nil {
			t.Fatalf("merge %s: %v", mod.ID, err)
		}
	}
	if normalizeYAML(cur) != normalizeYAML(content) {
		t.Fatal("round-trip yaml mismatch")
	}
}

func TestMergeConfigModuleUpdatesWAF(t *testing.T) {
	base := "version: v1\nport: 8080\nwaf:\n  enabled: true\n"
	patch := "waf:\n  enabled: false\n  log_only: true\n"
	out, err := MergeConfigModule(base, "waf", patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "log_only: true") {
		t.Fatalf("expected waf patch applied: %q", out)
	}
	if strings.Contains(out, "enabled: true") {
		t.Fatalf("expected old waf value replaced: %q", out)
	}
}

func TestChangedConfigModules(t *testing.T) {
	a := "version: v1\nport: 8080\nwaf:\n  enabled: true\n"
	b := "version: v1\nport: 9090\nwaf:\n  enabled: true\n"
	changed, err := ChangedConfigModules(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != "general" {
		t.Fatalf("changed=%v", changed)
	}
}

func TestMergeConfigModulePreservesKeyOrder(t *testing.T) {
	base := "version: v1\nport: 8080\n\n# Shared cache engine\ncache:\n  ttl: 300\n  host: 127.0.0.1\n"
	patch := "version: v1\nport: 8080\nenable_h2c: true\n"
	out, err := MergeConfigModule(base, "general", patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "enable_h2c: true") {
		t.Fatalf("expected enable_h2c: %q", out)
	}
	cacheIdx := strings.Index(out, "cache:")
	h2cIdx := strings.Index(out, "enable_h2c:")
	if h2cIdx < 0 || cacheIdx < 0 || h2cIdx > cacheIdx {
		t.Fatalf("expected enable_h2c before cache, got:\n%s", out)
	}
	// cache block should keep original node (ttl still present right after cache:)
	if !strings.Contains(out, "ttl: 300") {
		t.Fatalf("expected cache content preserved: %q", out)
	}
}
