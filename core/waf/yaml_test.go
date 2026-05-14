package waf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestApplyRulePatchesFromYAML_MapsPerRuleIndex(t *testing.T) {
	t.Parallel()
	raw := `
waf:
  enabled: true
rules:
  - host: a.example.com
    backend:
      service:
        name: httpbin.org
        port: 443
        protocol: https
  - host: b.example.com
    waf:
      trust_proxy: true
    backend:
      service:
        name: httpbin.org
        port: 443
        protocol: https
`
	rules := []rule.Rule{
		{Host: "a.example.com"},
		{Host: "b.example.com"},
	}
	if err := ApplyRulePatchesFromYAML([]byte(strings.TrimSpace(raw)), rules); err != nil {
		t.Fatal(err)
	}
	if rules[0].WAFPatch != nil {
		t.Fatalf("unexpected patch on rule0: %#v", rules[0].WAFPatch)
	}
	if rules[1].WAFPatch == nil {
		t.Fatal("expected patch on rule1")
	}
	b, ok := rules[1].WAFPatch["trust_proxy"].(bool)
	if !ok || !b {
		t.Fatalf("trust_proxy patch: %+v", rules[1].WAFPatch)
	}
}

func TestApplyRulePatchesFromYAML_WAFNotMapping_Error(t *testing.T) {
	t.Parallel()
	raw := `rules:
  - waf: not-a-map
`
	err := ApplyRulePatchesFromYAML([]byte(strings.TrimSpace(raw)), []rule.Rule{{}})
	if err == nil {
		t.Fatal("expected error when .waf is not a mapping")
	}
}

func TestApplyRulePatchesFromFile_success_and_readError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	content := "rules:\n  - waf:\n      trust_proxy: true\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	rules := []rule.Rule{{}}
	if err := ApplyRulePatchesFromFile(path, rules); err != nil {
		t.Fatal(err)
	}
	if rules[0].WAFPatch == nil {
		t.Fatal("expected patch")
	}

	err := ApplyRulePatchesFromFile(filepath.Join(dir, "missing.yaml"), []rule.Rule{})
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected ErrNotExist wrap/read err: %v", err)
	}
}

func TestApplyRulePatchesFromYAML_invalidYAML(t *testing.T) {
	t.Parallel()
	err := ApplyRulePatchesFromYAML([]byte("rules: [\n"), []rule.Rule{{}})
	if err == nil {
		t.Fatal("invalid yaml expected error")
	}
}

func TestApplyRulePatchesFromYAML_rulesNotSlice_clearsPatches(t *testing.T) {
	t.Parallel()
	rules := []rule.Rule{{WAFPatch: map[string]any{"x": true}}}
	raw := `rules: notalist`
	if err := ApplyRulePatchesFromYAML([]byte(strings.TrimSpace(raw)), rules); err != nil {
		t.Fatal(err)
	}
	if rules[0].WAFPatch != nil {
		t.Fatal("unsupported rules shape should yield nil WAFPatch on rule0")
	}
}

func TestApplyRulePatchesFromYAML_ruleScalar_skips_patch(t *testing.T) {
	t.Parallel()
	raw := `rules:
  - hello
`
	rules := []rule.Rule{{}}
	if err := ApplyRulePatchesFromYAML([]byte(strings.TrimSpace(raw)), rules); err != nil {
		t.Fatal(err)
	}
	if rules[0].WAFPatch != nil {
		t.Fatalf("%#v", rules[0].WAFPatch)
	}
}

func TestApplyRulePatchesFromYAML_resetsStalePatch_indexOutOfYAML(t *testing.T) {
	t.Parallel()
	raw := `rules: []`
	rules := []rule.Rule{{WAFPatch: map[string]any{"stale": true}}}
	if err := ApplyRulePatchesFromYAML([]byte(strings.TrimSpace(raw)), rules); err != nil {
		t.Fatal(err)
	}
	if rules[0].WAFPatch != nil {
		t.Fatal("YAML shorter than cfg rules should reset patch slots")
	}
}
