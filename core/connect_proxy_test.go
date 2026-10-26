package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/go-zoox/config"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestCompileConnectProxyRules(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "proxy.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name: "http-proxy",
						Port: 3128,
					},
				},
				ConnectProxy: rule.ConnectProxy{
					Enabled:        true,
					AllowedTargets: []string{`(^|\.)example\.com:443$`},
				},
			},
			{
				Host: "disabled.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name: "disabled-proxy",
						Port: 3128,
					},
				},
			},
		},
	}

	compiled, err := compileConnectProxyRules(cfg)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(compiled) != 1 || compiled[0].entry.ruleIndex != 0 {
		t.Fatalf("unexpected compiled rules: %+v", compiled)
	}
	if !targetAllowed(compiled[0].targetPatterns, "example.com:443") {
		t.Fatal("expected target to be allowed")
	}
	if targetAllowed(compiled[0].targetPatterns, "example.org:443") {
		t.Fatal("expected target to be denied")
	}
}

func TestCompileConnectProxyRulesRejectsInvalidTargetPattern(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "proxy.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name: "http-proxy",
					},
				},
				ConnectProxy: rule.ConnectProxy{
					Enabled:        true,
					AllowedTargets: []string{"(["},
				},
			},
		},
	}

	if _, err := compileConnectProxyRules(cfg); err == nil {
		t.Fatal("expected invalid target pattern error")
	}
}

func TestMatchConnectProxyUsesSNI(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host:     "proxy.example.com",
				HostType: "exact",
				Backend: rule.Backend{
					Service: service.Service{
						Name: "http-proxy",
						Port: 3128,
					},
				},
				ConnectProxy: rule.ConnectProxy{Enabled: true},
			},
		},
	}

	c := &core{cfg: cfg}
	router, err := compileRouterIndex(cfg.Rules, rule.Backend{})
	if err != nil {
		t.Fatalf("compile router: %v", err)
	}
	c.router = router
	compiled, err := compileConnectProxyRules(cfg)
	if err != nil {
		t.Fatalf("compile connect rules: %v", err)
	}
	c.connectProxyRules = compiled

	serviceCfg, ruleCfg, compiledRule, err := c.matchConnectProxy("proxy.example.com")
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if serviceCfg == nil || serviceCfg.Target() != "http://http-proxy:3128" || ruleCfg.Host != "proxy.example.com" || compiledRule.entry.ruleIndex != 0 {
		t.Fatalf("unexpected match: %+v / %+v / %+v", serviceCfg, ruleCfg, compiledRule)
	}
	if _, _, _, err := c.matchConnectProxy("other.example.com"); err == nil {
		t.Fatal("expected non-proxy SNI to be denied")
	}
}

func TestProxyRequestAuthority(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"http://example.com/path", "example.com:80"},
		{"http://example.com:8080/path", "example.com:8080"},
		{"http://EXAMPLE.com", "example.com:80"},
	}
	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodGet, tc.raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := proxyRequestAuthority(req); got != tc.want {
			t.Fatalf("proxyRequestAuthority(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestConnectHTTPProxyTransportPreservesAbsoluteURIAndProxyAuth(t *testing.T) {
	var gotRequestURI, gotHost, gotAuth string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestURI = r.RequestURI
		gotHost = r.Host
		gotAuth = r.Header.Get("Proxy-Authorization")
		w.Header().Set("Proxy-Authenticate", `Basic realm="test"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse("http://target.example.test/path?q=1")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Proxy-Authorization", "Basic abc")
	req = req.WithContext(context.WithValue(req.Context(), connectProxyURLKey{}, proxyURL))

	transport := newConnectHTTPProxyTransport()
	defer transport.CloseIdleConnections()
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	if gotRequestURI != "http://target.example.test/path?q=1" {
		t.Fatalf("proxy received URI %q", gotRequestURI)
	}
	if gotHost != "target.example.test" {
		t.Fatalf("proxy received Host %q", gotHost)
	}
	if gotAuth != "Basic abc" {
		t.Fatalf("proxy received Proxy-Authorization %q", gotAuth)
	}
}

func TestCopyProxyResponseHeaders(t *testing.T) {
	src := http.Header{
		"Connection":         []string{"keep-alive"},
		"Content-Type":       []string{"text/plain"},
		"Proxy-Authenticate": []string{`Basic realm="test"`},
	}
	dst := http.Header{}
	copyProxyResponseHeaders(dst, src)
	if dst.Get("Connection") != "" {
		t.Fatal("hop-by-hop Connection header should be removed")
	}
	if dst.Get("Proxy-Authenticate") == "" {
		t.Fatal("Proxy-Authenticate should be preserved")
	}
	if dst.Get("Content-Type") != "text/plain" {
		t.Fatal("end-to-end header should be preserved")
	}
}

func TestTargetAllowedUsesWholeMatch(t *testing.T) {
	patterns := []*regexp.Regexp{regexp.MustCompile(`example\.com:443`)}
	if targetAllowed(patterns, "evil-example.com:443") {
		t.Fatal("substring must not satisfy target ACL")
	}
	if !targetAllowed(patterns, "example.com:443") {
		t.Fatal("full target should match")
	}
}

func TestConnectProxyRouteIsNotShadowedByNormalRule(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "*.yss.example.com",
				Backend: rule.Backend{Service: service.Service{
					Name: "normal",
					Port: 8080,
				}},
			},
			{
				Host: "anchor.yss.example.com",
				Backend: rule.Backend{Service: service.Service{
					Name: "fallback-normal",
					Port: 8080,
				}},
				ConnectProxy: rule.ConnectProxy{
					Enabled: true,
					Upstream: service.Service{
						Name:     "anchor-proxy",
						Port:     3128,
						Protocol: "http",
					},
				},
			},
		},
	}
	if err := inferBackendTypes(cfg); err != nil {
		t.Fatal(err)
	}
	c := &core{cfg: cfg}
	router, err := compileRouterIndex(cfg.Rules, cfg.Fallback)
	if err != nil {
		t.Fatal(err)
	}
	c.router = router
	compiled, err := compileConnectProxyRules(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.connectProxyRules = compiled

	fallback, ruleCfg, _, err := c.matchConnectProxy("anchor.yss.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if ruleCfg.Host != "anchor.yss.example.com" {
		t.Fatalf("matched %q, want exact connect rule", ruleCfg.Host)
	}
	upstream, err := connectProxyUpstream(ruleCfg, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if upstream.Target() != "http://anchor-proxy:3128" {
		t.Fatalf("upstream = %s", upstream.Target())
	}
}

func TestConnectProxyUpstreamFallsBackToBackend(t *testing.T) {
	ruleCfg := &rule.Rule{Backend: rule.Backend{Service: service.Service{Name: "anchor-backend", Port: 3128}}}
	fallback := &ruleCfg.Backend.Service
	upstream, err := connectProxyUpstream(ruleCfg, fallback)
	if err != nil {
		t.Fatal(err)
	}
	if upstream.Target() != "http://anchor-backend:3128" {
		t.Fatalf("upstream = %s", upstream.Target())
	}
}

func TestConnectProxyMaxConnections(t *testing.T) {
	compiled := &compiledConnectProxyRule{maxConnections: 2}
	if !compiled.acquire() || !compiled.acquire() {
		t.Fatal("first two connections should be allowed")
	}
	if compiled.acquire() {
		t.Fatal("third connection should be rejected")
	}
	compiled.release()
	if !compiled.acquire() {
		t.Fatal("connection slot should be reusable")
	}
	compiled.release()
	compiled.release()
}

func TestConfigLoadConnectProxyAppliesEveryUIValue(t *testing.T) {
	raw := []byte(`
version: v1
port: 8080
https:
  port: 8443
  ssl:
    - domain: proxy.example.com
      cert:
        certificate: /tmp/proxy.crt
        certificate_key: /tmp/proxy.key
rules:
  - host: proxy.example.com
    backend:
      service:
        name: anchor-frontend
        port: 8080
    connect_proxy:
      enabled: true
      allowed_targets:
        - '^.+:443$'
        - '^db\.internal:5432$'
      upstream:
        name: anchor-backend
        port: 3128
        protocol: http
      connect_timeout: 7
      response_timeout: 9
      idle_timeout: 120
      max_connections: 3
`)

	tmp, err := os.CreateTemp("", "ingress-connect-proxy-ui-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}

	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: tmp.Name()}); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if err := ValidateConfig(&cfg); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	compiled, err := compileConnectProxyRules(&cfg)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(compiled) != 1 {
		t.Fatalf("compiled = %d, want 1", len(compiled))
	}

	cp := cfg.Rules[0].ConnectProxy
	if !cp.Enabled {
		t.Fatal("enabled was not loaded")
	}
	if cp.Upstream.Name != "anchor-backend" || cp.Upstream.Port != 3128 || cp.Upstream.Protocol != "http" {
		t.Fatalf("upstream = %+v", cp.Upstream)
	}
	if cp.ConnectTimeout != 7 || cp.ResponseTimeout != 9 || cp.IdleTimeout != 120 || cp.MaxConnections != 3 {
		t.Fatalf("limits = %+v", cp)
	}
	if len(compiled[0].targetPatterns) != 2 {
		t.Fatalf("target patterns = %d", len(compiled[0].targetPatterns))
	}
	if !targetAllowed(compiled[0].targetPatterns, "example.com:443") {
		t.Fatal("HTTPS target should be allowed")
	}
	if !targetAllowed(compiled[0].targetPatterns, "db.internal:5432") {
		t.Fatal("explicit target should be allowed")
	}
	if compiled[0].connectTimeout != 7*time.Second || compiled[0].responseTimeout != 9*time.Second || compiled[0].idleTimeout != 120*time.Second || compiled[0].maxConnections != 3 {
		t.Fatalf("compiled limits = connect=%s response=%s idle=%s max=%d",
			compiled[0].connectTimeout,
			compiled[0].responseTimeout,
			compiled[0].idleTimeout,
			compiled[0].maxConnections,
		)
	}
}

func TestConnectProxyDefaultLimits(t *testing.T) {
	cfg := &Config{Rules: []rule.Rule{{
		Host:     "proxy.example.com",
		HostType: "exact",
		Backend: rule.Backend{Service: service.Service{
			Name: "proxy",
			Port: 3128,
		}},
		ConnectProxy: rule.ConnectProxy{Enabled: true},
	}}}
	if err := inferBackendTypes(cfg); err != nil {
		t.Fatal(err)
	}
	compiled, err := compileConnectProxyRules(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if compiled[0].connectTimeout != defaultConnectProxyConnectTimeout {
		t.Fatalf("connect timeout = %s", compiled[0].connectTimeout)
	}
	if compiled[0].responseTimeout != defaultConnectProxyResponseTimeout {
		t.Fatalf("response timeout = %s", compiled[0].responseTimeout)
	}
	if compiled[0].idleTimeout != defaultConnectProxyIdleTimeout {
		t.Fatalf("idle timeout = %s", compiled[0].idleTimeout)
	}
	if compiled[0].maxConnections != 0 {
		t.Fatalf("max connections = %d", compiled[0].maxConnections)
	}
}

func TestConnectProxyUIValuesAreValidated(t *testing.T) {
	cfg := &Config{
		HTTPS: HTTPS{
			Port: 8443,
			SSL:  []SSL{{Domain: "proxy.example.com", Cert: SSLCert{Certificate: "/tmp/c", CertificateKey: "/tmp/k"}}},
		},
		Rules: []rule.Rule{{
			Host: "proxy.example.com",
			Backend: rule.Backend{Service: service.Service{
				Name: "proxy",
				Port: 3128,
			}},
			ConnectProxy: rule.ConnectProxy{
				Enabled:        true,
				ConnectTimeout: -1,
			},
		}},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("negative timeout must fail validation")
	}
}
