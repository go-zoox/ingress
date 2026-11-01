package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func newIPAllowlistTestCore(t *testing.T, cfg *Config) *core {
	t.Helper()
	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}
	ins, ok := c.(*core)
	if !ok {
		t.Fatal("failed to cast core instance")
	}
	if err := ins.build(); err != nil {
		t.Fatalf("failed to build core: %v", err)
	}
	return ins
}

func serveIPAllowlistRequest(ins *core, host, remoteAddr, xff string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "http://"+host+"/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)
	return rec
}

func TestBuild_RouteIPAllowlist_BlocksOutsideAllow(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "admin.example.com",
				WAFPatch: map[string]any{
					"enabled":         true,
					"disable_builtin": true,
					"allow":           []any{"10.0.0.0/8"},
				},
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "ok"},
				},
			},
		},
	}
	ins := newIPAllowlistTestCore(t, cfg)

	allowed := serveIPAllowlistRequest(ins, "admin.example.com", "10.1.2.3:1234", "")
	if allowed.Code != 200 || allowed.Body.String() != "ok" {
		t.Fatalf("allowlisted source: got %d %q", allowed.Code, allowed.Body.String())
	}

	blocked := serveIPAllowlistRequest(ins, "admin.example.com", "198.51.100.23:1234", "")
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("non-allowlisted source: expected 403, got %d", blocked.Code)
	}
}

func TestBuild_RouteIPAllowlist_TrustedProxyIgnoresSpoofedXFF(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		WAF:  rule.WAF{TrustedProxies: []string{"10.0.0.0/8"}},
		Rules: []rule.Rule{
			{
				Host: "internal.example.com",
				WAFPatch: map[string]any{
					"enabled":         true,
					"disable_builtin": true,
					"allow":           []any{"203.0.113.0/24"},
				},
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "ok"},
				},
			},
		},
	}
	ins := newIPAllowlistTestCore(t, cfg)

	// Peer is a trusted proxy: rightmost non-trusted XFF hop is the real client.
	viaProxy := serveIPAllowlistRequest(ins, "internal.example.com", "10.0.0.5:1234", "203.0.113.7, 10.0.0.2")
	if viaProxy.Code != 200 {
		t.Fatalf("trusted proxy should resolve 203.0.113.7, got %d", viaProxy.Code)
	}

	// Peer is not a trusted proxy: a forged XFF must not bypass the allowlist.
	spoofed := serveIPAllowlistRequest(ins, "internal.example.com", "198.51.100.23:1234", "203.0.113.7")
	if spoofed.Code != http.StatusForbidden {
		t.Fatalf("spoofed XFF must be ignored, expected 403, got %d", spoofed.Code)
	}
}

func TestBuild_RouteIPAllowlist_DisabledByDefault(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "public.example.com",
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{StatusCode: 200, Body: "ok"},
				},
			},
		},
	}
	ins := newIPAllowlistTestCore(t, cfg)

	rec := serveIPAllowlistRequest(ins, "public.example.com", "198.51.100.23:1234", "")
	if rec.Code != 200 {
		t.Fatalf("route without allowlist must stay open, got %d", rec.Code)
	}
}
