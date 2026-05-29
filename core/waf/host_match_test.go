package waf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCheckRequest_AllowHosts_SkipsWAF(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		AllowHosts:     []string{"admin.internal", "*.cdn.example.com"},
		Deny:           []string{"127.0.0.1"},
		Rules: []rule.WAFRule{{
			ID:      "block-all",
			Type:    PatternTypeContains,
			Pattern: "/",
			Targets: []string{TargetPath},
		}},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://admin.internal/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, "admin.internal", "/", http.MethodGet, nil) {
		t.Fatal("allow_hosts exact match must skip WAF")
	}

	req2 := httptest.NewRequest(http.MethodGet, "http://assets.cdn.example.com/", nil)
	req2.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req2, "assets.cdn.example.com", "/", http.MethodGet, nil) {
		t.Fatal("allow_hosts wildcard must skip WAF")
	}

	req3 := httptest.NewRequest(http.MethodGet, "http://api.example.com/", nil)
	req3.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req3, "api.example.com", "/", http.MethodGet, nil) {
		t.Fatal("non-whitelisted host must still be blocked")
	}
}

func TestCompileProfile_AllowHosts_InvalidPattern(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled:    true,
		AllowHosts: []string{"("},
	})
	if err == nil {
		t.Fatal("expected invalid allow_hosts regex error")
	}
}

func TestCompileProfile_RuleAllowHosts_InvalidPattern(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{{
			ID:         "r1",
			Type:       PatternTypeContains,
			Pattern:    "x",
			Targets:    []string{TargetPath},
			AllowHosts: []string{"("},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "allow_hosts") {
		t.Fatalf("err=%v", err)
	}
}

func TestCheckRequest_RuleAllowHosts_BypassesRuleOnly(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{
			{
				ID:         "scoped",
				Type:       PatternTypeContains,
				Pattern:    "BAD",
				Targets:    []string{TargetPath},
				AllowHosts: []string{"safe.example.com"},
			},
			{
				ID:      "other",
				Type:    PatternTypeContains,
				Pattern: "NEEDLE",
				Targets: []string{TargetPath},
			},
		},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://safe.example.com/BAD", nil)
	req.RemoteAddr = "127.0.0.1:1"
	if CheckRequest(prof, req, "safe.example.com", "/BAD", http.MethodGet, nil) {
		t.Fatal("scoped rule allow_hosts should bypass that rule; NEEDLE rule should not match BAD")
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://other.example.com/BAD", nil)
	req2.RemoteAddr = "127.0.0.1:1"
	if !CheckRequest(prof, req2, "other.example.com", "/BAD", http.MethodGet, nil) {
		t.Fatal("scoped rule should still block on non-whitelisted hosts")
	}
}

func TestHostMatchesAllowList_PortStripped(t *testing.T) {
	t.Parallel()
	m, err := compileHostPattern("api.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !hostMatchesAllowList("api.example.com:8080", []hostMatcher{m}) {
		t.Fatal("expected host:port to match exact host pattern")
	}
}
