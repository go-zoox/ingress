package waf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCompileTrustedProxies_Validation(t *testing.T) {
	t.Parallel()

	t.Run("valid CIDRs and bare IPs", func(t *testing.T) {
		nets, err := compileTrustedProxyNets([]string{"10.0.0.0/8", "203.0.113.7", "  ", "2001:db8::/32"}, "waf.trusted_proxies")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(nets) != 3 {
			t.Fatalf("expected 3 nets, got %d", len(nets))
		}
	})

	t.Run("invalid CIDR", func(t *testing.T) {
		_, err := compileTrustedProxyNets([]string{"10.0.0.0/33"}, "waf.trusted_proxies")
		if err == nil || !strings.Contains(err.Error(), "invalid CIDR") {
			t.Fatalf("expected invalid CIDR error, got %v", err)
		}
	})

	t.Run("all-addresses v4 rejected", func(t *testing.T) {
		_, err := compileTrustedProxyNets([]string{"0.0.0.0/0"}, "waf.trusted_proxies")
		if err == nil || !strings.Contains(err.Error(), "trusts every source") {
			t.Fatalf("expected /0 rejection, got %v", err)
		}
	})

	t.Run("all-addresses v6 rejected", func(t *testing.T) {
		_, err := compileTrustedProxyNets([]string{"::/0"}, "waf.trusted_proxies")
		if err == nil || !strings.Contains(err.Error(), "trusts every source") {
			t.Fatalf("expected /0 rejection, got %v", err)
		}
	})
}

func TestCompileIngress_ValidatesGlobalTrustedProxiesWhenDisabled(t *testing.T) {
	t.Parallel()
	// A malformed security setting must fail fast even while WAF is off.
	_, _, err := CompileIngress(rule.WAF{TrustedProxies: []string{"not-a-cidr"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "trusted_proxies") {
		t.Fatalf("expected trusted_proxies error, got %v", err)
	}
}

func TestMergePatch_TrustedProxies(t *testing.T) {
	t.Parallel()
	merged, err := MergePatch(rule.WAF{Enabled: true}, map[string]any{
		"trusted_proxies": []any{"10.0.0.0/8", "172.16.0.0/12"},
	}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(merged.TrustedProxies) != 2 || merged.TrustedProxies[0] != "10.0.0.0/8" {
		t.Fatalf("unexpected trusted proxies: %#v", merged.TrustedProxies)
	}
}

func TestClientIP_TrustedProxies(t *testing.T) {
	t.Parallel()
	nets, err := compileTrustedProxyNets([]string{"10.0.0.0/8", "172.16.0.0/12"}, "waf.trusted_proxies")
	if err != nil {
		t.Fatal(err)
	}

	newReq := func(remote, xff string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "http://x/", nil)
		r.RemoteAddr = remote
		if xff != "" {
			r.Header.Set(headerXForwardedFor, xff)
		}
		return r
	}

	t.Run("rightmost untrusted hop", func(t *testing.T) {
		got := clientIP(newReq("10.0.0.5:1234", "203.0.113.7, 10.0.0.2"), false, 0, nets)
		if got == nil || got.String() != "203.0.113.7" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("untrusted peer ignores XFF", func(t *testing.T) {
		got := clientIP(newReq("198.51.100.23:1234", "203.0.113.7"), false, 0, nets)
		if got == nil || got.String() != "198.51.100.23" {
			t.Fatalf("forged XFF must be ignored, got %v", got)
		}
	})

	t.Run("all hops trusted falls back to peer", func(t *testing.T) {
		got := clientIP(newReq("10.0.0.5:1234", "10.0.0.9, 10.0.0.2"), false, 0, nets)
		if got == nil || got.String() != "10.0.0.5" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("empty XFF uses peer", func(t *testing.T) {
		got := clientIP(newReq("10.0.0.5:1234", ""), false, 0, nets)
		if got == nil || got.String() != "10.0.0.5" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("invalid XFF segments are skipped", func(t *testing.T) {
		got := clientIP(newReq("10.0.0.5:1234", "garbage, 203.0.113.7"), false, 0, nets)
		if got == nil || got.String() != "203.0.113.7" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("IPv6 trusted proxy", func(t *testing.T) {
		v6nets, err := compileTrustedProxyNets([]string{"2001:db8::/32"}, "waf.trusted_proxies")
		if err != nil {
			t.Fatal(err)
		}
		got := clientIP(newReq("[2001:db8::1]:443", "203.0.113.9"), false, 0, v6nets)
		if got == nil || got.String() != "203.0.113.9" {
			t.Fatalf("got %v", got)
		}
	})
}

func TestCheckRequest_TrustedProxies_AllowGate(t *testing.T) {
	t.Parallel()
	m := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		TrustedProxies: []string{"10.0.0.0/8"},
		Allow:          []string{"203.0.113.0/24"},
	}
	prof, err := compileProfile(0, m)
	if err != nil {
		t.Fatal(err)
	}

	allowed := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	allowed.RemoteAddr = "10.0.0.5:1234"
	allowed.Header.Set(headerXForwardedFor, "203.0.113.7, 10.0.0.2")
	if CheckRequest(prof, allowed, "x", "/", http.MethodGet, nil) {
		t.Fatal("trusted proxy hop should resolve to allowed client 203.0.113.7")
	}

	spoofed := httptest.NewRequest(http.MethodGet, "http://x/", nil)
	spoofed.RemoteAddr = "198.51.100.23:1234"
	spoofed.Header.Set(headerXForwardedFor, "203.0.113.7")
	if !CheckRequest(prof, spoofed, "x", "/", http.MethodGet, nil) {
		t.Fatal("untrusted peer must be blocked even when XFF claims an allowed IP")
	}
}
