package waf

import (
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestBuiltinRuleEnabled_defaults(t *testing.T) {
	t.Parallel()
	if !BuiltinRuleEnabled(false, nil, "builtin:xss-lite") {
		t.Fatal("expected enabled when disable_builtin false")
	}
	if BuiltinRuleEnabled(true, nil, "builtin:xss-lite") {
		t.Fatal("expected disabled when disable_builtin true")
	}
}

func TestBuiltinRuleEnabled_override(t *testing.T) {
	t.Parallel()
	overrides := map[string]bool{"builtin:xss-lite": false}
	if BuiltinRuleEnabled(false, overrides, "builtin:xss-lite") {
		t.Fatal("override false should disable")
	}
	overrides["builtin:xss-lite"] = true
	if !BuiltinRuleEnabled(true, overrides, "builtin:xss-lite") {
		t.Fatal("override true should enable even when disable_builtin true")
	}
}

func TestRuleActive_customEnabled(t *testing.T) {
	t.Parallel()
	on := true
	off := false
	if !RuleActive(rule.WAFRule{ID: "a", Enabled: nil}) {
		t.Fatal("nil enabled defaults true")
	}
	if RuleActive(rule.WAFRule{ID: "a", Enabled: &off}) {
		t.Fatal("explicit false disables")
	}
	if !RuleActive(rule.WAFRule{ID: "a", Enabled: &on}) {
		t.Fatal("explicit true enables")
	}
}

func TestCompileProfile_builtinRulesSelective(t *testing.T) {
	t.Parallel()
	disabled := false
	prof, err := compileProfile(0, rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		BuiltinRules: map[string]bool{
			"builtin:xss-lite": true,
		},
		Rules: []rule.WAFRule{{
			ID: "custom-off", Type: PatternTypeContains, Pattern: "OFF", Targets: []string{TargetQuery},
			Enabled: &disabled,
		}, {
			ID: "custom-on", Type: PatternTypeContains, Pattern: "ON", Targets: []string{TargetQuery},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(prof.signatureRules) != 2 {
		t.Fatalf("rules=%d want xss-lite + custom-on", len(prof.signatureRules))
	}
	req := httptest.NewRequest("GET", "http://x/p?q=ON", nil)
	if !CheckRequest(prof, req, "x", "/p", "GET", nil) {
		t.Fatal("custom-on should block")
	}
	req2 := httptest.NewRequest("GET", "http://x/p?q=OFF", nil)
	if CheckRequest(prof, req2, "x", "/p", "GET", nil) {
		t.Fatal("custom-off should not block")
	}
	req3 := httptest.NewRequest("GET", "http://x/p?q=<script>x", nil)
	if !CheckRequest(prof, req3, "x", "/p", "GET", nil) {
		t.Fatal("enabled builtin xss-lite should block")
	}
}

func TestStarterRules_compileAll(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{Enabled: true, DisableBuiltin: false})
	if err != nil {
		t.Fatal(err)
	}
}
