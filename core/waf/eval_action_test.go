package waf

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCheckRequest_RuleActionPass_StopsChain(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{
			{
				ID:      "allow-token",
				Action:  ActionPass,
				Type:    PatternTypeContains,
				Pattern: "SAFE",
				Targets: []string{TargetQuery},
			},
			{
				ID:      "block-secret",
				Type:    PatternTypeContains,
				Pattern: "SECRET",
				Targets: []string{TargetQuery},
			},
		},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?q=SAFE+SECRET", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, "x", "/p", http.MethodGet, nil) {
		t.Fatal("pass rule must stop evaluation before block rule")
	}
}

func TestCheckRequest_RuleActionAudit_ContinuesToBlock(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{
			{
				ID:      "audit-hit",
				Action:  ActionAudit,
				Type:    PatternTypeContains,
				Pattern: "AUDIT",
				Targets: []string{TargetQuery},
			},
			{
				ID:      "block-hit",
				Type:    PatternTypeContains,
				Pattern: "BLOCK",
				Targets: []string{TargetQuery},
			},
		},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?q=AUDIT+BLOCK", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/p", http.MethodGet, nil) {
		t.Fatal("audit must not block; later block rule should match")
	}
}

func TestCheckRequest_BuiltinRuleActionAudit(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled: true,
		BuiltinRuleActions: map[string]string{
			"builtin:xss-lite": ActionAudit,
		},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?q=%3Cscript%3Ealert(1)", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, "x", "/p", http.MethodGet, nil) {
		t.Fatal("builtin:xss-lite with audit action must not block")
	}
}

func TestCheckRequest_ExplicitBlockOverridesGlobalLogOnly(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		LogOnly:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:      "still-block",
			Action:  ActionBlock,
			Type:    PatternTypeContains,
			Pattern: "DENY",
			Targets: []string{TargetQuery},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/p?q=DENY", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req, "x", "/p", http.MethodGet, nil) {
		t.Fatal("per-rule block must override global log_only")
	}
}
