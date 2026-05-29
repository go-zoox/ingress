package waf

import (
	"net/http"
	"net/http/httptest"
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
