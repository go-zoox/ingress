package service

import (
	"testing"

	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	coresvc "github.com/go-zoox/ingress/core/service"
)

func TestParseServiceCatalog_and_ListServiceRouteRefs(t *testing.T) {
	yaml := `
services:
  - name: api.internal
    port: 8080
    protocol: http
    healthcheck:
      enable: true
      path: /health
rules:
  - host: api.example.com
    backend:
      service:
        name: api.internal
        port: 8080
`
	catalog, err := ParseServiceCatalog(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != 1 {
		t.Fatalf("catalog len=%d", len(catalog))
	}
	if catalog[0].Target != "api.internal:8080" {
		t.Fatalf("target=%q", catalog[0].Target)
	}
	if !catalog[0].HealthCheck.Enable {
		t.Fatal("expected healthcheck")
	}

	cfg := &ingcore.Config{
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Type: "service",
					Service: coresvc.Service{Name: "api.internal", Port: 8080},
				},
			},
		},
	}
	refs := ListServiceRouteRefs(cfg, "api.internal")
	if len(refs) != 1 || refs[0].PathIndex != -1 {
		t.Fatalf("refs=%+v", refs)
	}

	entries := []AccessEntry{
		{Host: "api.example.com", Target: "api.internal:8080", Path: "/v1", Status: 200, DurationMs: 10},
		{Host: "other.example.com", Target: "other:8080", Path: "/", Status: 200, DurationMs: 5},
	}
	filtered := FilterAccessEntriesForService(entries, ServiceTargetAliases(catalog[0], refs))
	if len(filtered) != 1 {
		t.Fatalf("filtered=%d", len(filtered))
	}
}

func TestFindCatalogService_notFound(t *testing.T) {
	_, ok := FindCatalogService(nil, "missing")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestServiceTarget_defaultPort(t *testing.T) {
	if got := ServiceTarget("x", 0, "https"); got != "x:443" {
		t.Fatalf("got %q", got)
	}
	if got := ServiceTarget("x", 0, ""); got != "x:80" {
		t.Fatalf("got %q", got)
	}
}

func TestFilterAccessEntriesForService_empty(t *testing.T) {
	out := FilterAccessEntriesForService([]AccessEntry{{Target: "a:1"}}, nil)
	if len(out) != 0 {
		t.Fatalf("len=%d", len(out))
	}
}
