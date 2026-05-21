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
