package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestListRouteRows_and_PreviewMatch(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Service: service.Service{Name: "backend.internal", Port: 8080},
				},
			},
		},
	}
	rows, err := ListRouteRows(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: got %d want 1", len(rows))
	}
	if rows[0].HostType != "exact" {
		t.Fatalf("host_type: %s", rows[0].HostType)
	}

	preview, err := PreviewMatch(cfg, "api.example.com", "/v1")
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Matched {
		t.Fatalf("expected match: %+v", preview)
	}
}

func TestRequestMatchesRoute_regexHost(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: `mp-infra-([^.]+)\.example\.com`,
				Backend: rule.Backend{
					Service: service.Service{Name: "$1.backend.internal", Port: 8080},
				},
			},
		},
	}
	ok, err := RequestMatchesRoute(cfg, 0, -1, "mp-infra-foo.example.com", "/")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected mp-infra-foo.example.com to match rule 0")
	}
	ok, err = RequestMatchesRoute(cfg, 0, -1, "other.example.com", "/")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected other.example.com not to match rule 0")
	}
	ok, err = RequestMatchesRoute(cfg, 0, -1, `mp-infra-([^.]+)\.example\.com`, "/")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("config host pattern must not match as request host")
	}
}

func TestRequestMatchesRoute_ruleLevelAllHostTraffic(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Service: service.Service{Name: "api.internal", Port: 8080},
				},
				Paths: []rule.Path{
					{
						Path: "/v2",
						Backend: rule.Backend{
							Service: service.Service{Name: "api-v2.internal", Port: 8080},
						},
					},
				},
			},
		},
	}
	ok, err := RequestMatchesRoute(cfg, 0, -1, "api.example.com", "/api/users")
	if err != nil || !ok {
		t.Fatalf("rule-level /api/users: ok=%v err=%v", ok, err)
	}
	ok, err = RequestMatchesRoute(cfg, 0, -1, "api.example.com", "/v2/health")
	if err != nil || !ok {
		t.Fatalf("rule-level /v2/health: ok=%v err=%v", ok, err)
	}
	ok, err = RequestMatchesRoute(cfg, 0, 0, "api.example.com", "/v2/health")
	if err != nil || !ok {
		t.Fatalf("path 0 /v2/health: ok=%v err=%v", ok, err)
	}
	ok, err = RequestMatchesRoute(cfg, 0, 0, "api.example.com", "/api/users")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("path 0 should not match /api/users")
	}
}
