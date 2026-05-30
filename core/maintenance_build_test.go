package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func mustBuildIngressCore(t *testing.T, cfg *Config) *core {
	t.Helper()
	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ins, ok := c.(*core)
	if !ok {
		t.Fatal("expected *core")
	}
	if err := ins.build(); err != nil {
		t.Fatalf("build: %v", err)
	}
	return ins
}

func TestBuild_GlobalMaintenance_Returns503(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			Title: "Planned downtime",
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/api", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get(headerXIngressMaintenance); got != ingressMaintenanceHeaderVal {
		t.Fatalf("expected %s: %s, got %q", headerXIngressMaintenance, ingressMaintenanceHeaderVal, got)
	}
	if !strings.Contains(rec.Body.String(), "Planned downtime") {
		t.Fatalf("expected maintenance title in body, got %q", rec.Body.String())
	}
}

func TestBuild_GlobalMaintenance_BypassPath_PassesToUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	host, port := parseTestUpstreamHostPort(t, upstream.URL)

	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			Title: "Planned downtime",
			Bypass: service.MaintenanceBypass{
				Paths: []string{"/healthz"},
			},
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     host,
						Port:     port,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)

	reqBypass := httptest.NewRequest(http.MethodGet, "http://app.example.com/healthz", nil)
	recBypass := httptest.NewRecorder()
	ins.app.ServeHTTP(recBypass, reqBypass)
	if recBypass.Code != http.StatusOK || recBypass.Body.String() != "upstream-ok" {
		t.Fatalf("expected bypass path to reach upstream, got %d body=%q", recBypass.Code, recBypass.Body.String())
	}

	reqBlock := httptest.NewRequest(http.MethodGet, "http://app.example.com/api", nil)
	recBlock := httptest.NewRecorder()
	ins.app.ServeHTTP(recBlock, reqBlock)
	if recBlock.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on non-bypass path, got %d", recBlock.Code)
	}
}

func TestBuild_RuleMaintenance_ScopeAll_Returns503WithTitle(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
						Maintenance: service.Maintenance{
							Enabled: true,
							Scope:   service.MaintenanceScopeAll,
							Title:   "Rule maintenance window",
						},
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Rule maintenance window") {
		t.Fatalf("expected rule title in body, got %q", rec.Body.String())
	}
}

func TestBuild_Maintenance_RetryAfterHeader(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts:      hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			RetryAfter: 300,
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") != "300" {
		t.Fatalf("expected Retry-After 300, got %q", rec.Header().Get("Retry-After"))
	}
}

func TestBuild_Maintenance_NotBlockedWhenHostOutsideGlobalList(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	host, port := parseTestUpstreamHostPort(t, upstream.URL)

	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{Host: "maint.example.com"}),
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     host,
						Port:     port,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected upstream 200 when host not in maintenance list, got %d", rec.Code)
	}
}

func TestBuild_Maintenance_CustomResponseHeader(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			ResponseHeader: service.MaintenanceResponseHeader{
				Name:  "X-Custom-Maintenance",
				Value: "yes",
			},
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/api", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Custom-Maintenance"); got != "yes" {
		t.Fatalf("expected custom maintenance header, got %q", got)
	}
	if rec.Header().Get(headerXIngressMaintenance) != "" {
		t.Fatalf("expected default header absent, got %q", rec.Header().Get(headerXIngressMaintenance))
	}
}

func TestBuild_IngressStatus_IncludesCustomHeaderInJSON(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			ResponseHeader: service.MaintenanceResponseHeader{
				Name:  "X-Custom-Maintenance",
				Value: "yes",
			},
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/_/ingress/status", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	var body ingressStatusBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.MaintenanceHeaderName != "X-Custom-Maintenance" || body.MaintenanceHeaderValue != "yes" {
		t.Fatalf("unexpected status body header fields: %+v", body)
	}
	if rec.Header().Get("X-Custom-Maintenance") != "yes" {
		t.Fatalf("expected custom header on status response")
	}
}

func TestBuild_IngressStatus_CustomPath(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			StatusPath: "/internal/ingress-status",
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)

	reqCustom := httptest.NewRequest(http.MethodGet, "http://app.example.com/internal/ingress-status", nil)
	recCustom := httptest.NewRecorder()
	ins.app.ServeHTTP(recCustom, reqCustom)
	if recCustom.Code != http.StatusOK {
		t.Fatalf("expected 200 on custom status path, got %d body=%q", recCustom.Code, recCustom.Body.String())
	}

	reqDefault := httptest.NewRequest(http.MethodGet, "http://app.example.com/_/ingress/status", nil)
	recDefault := httptest.NewRecorder()
	ins.app.ServeHTTP(recDefault, reqDefault)
	// Default path is not registered when status_path is customized.
	if recDefault.Code == http.StatusOK {
		var body ingressStatusBody
		if err := json.Unmarshal(recDefault.Body.Bytes(), &body); err == nil && body.Status == "ok" {
			t.Fatal("expected default status path to not be handled")
		}
	}
}

func TestBuild_IngressStatus_OK(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/_/ingress/status", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	var body ingressStatusBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %+v", body)
	}
	if rec.Header().Get(headerXIngressMaintenance) != "" {
		t.Fatalf("expected no maintenance header when ok, got %q", rec.Header().Get(headerXIngressMaintenance))
	}
}

func TestBuild_IngressStatus_Maintenance(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts:      hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			Title:      "Planned downtime",
			RetryAfter: 600,
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/_/ingress/status", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get(headerXIngressMaintenance); got != ingressMaintenanceHeaderVal {
		t.Fatalf("expected maintenance header on status endpoint, got %q", got)
	}
	if rec.Header().Get("Retry-After") != "600" {
		t.Fatalf("expected Retry-After 600, got %q", rec.Header().Get("Retry-After"))
	}
	var body ingressStatusBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "maintenance" || body.Title != "Planned downtime" || body.RetryAfter != 600 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestBuild_IngressStatus_IgnoresBypass(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Maintenance: MaintenanceConfig{
			Hosts: hosts(service.MaintenanceHostEntry{Host: "app.example.com"}),
			Bypass: service.MaintenanceBypass{
				Paths: []string{"/healthz"},
			},
		},
		Rules: []rule.Rule{
			{
				Host: "app.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
					Service: service.Service{
						Name:     "127.0.0.1",
						Port:     1,
						Protocol: "http",
					},
				},
			},
		},
	}

	ins := mustBuildIngressCore(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/_/ingress/status", nil)
	rec := httptest.NewRecorder()
	ins.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status endpoint to report maintenance regardless of bypass paths, got %d", rec.Code)
	}
}

func parseTestUpstreamHostPort(t *testing.T, rawURL string) (string, int64) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse upstream URL %q: %v", rawURL, err)
	}
	var port int64
	if _, err := fmt.Sscanf(u.Port(), "%d", &port); err != nil {
		t.Fatalf("parse upstream port from %q: %v", rawURL, err)
	}
	return u.Hostname(), port
}
