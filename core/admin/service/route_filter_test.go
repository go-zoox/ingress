package service

import (
	"testing"

	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestFilterAccessLinesForRoute_regexHost(t *testing.T) {
	cfg := &ingcore.Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: `api-([a-z]+)\.example\.com`,
				Backend: rule.Backend{
					Service: service.Service{Name: "upstream", Port: 8080},
				},
			},
		},
	}
	lines := []string{
		`1.2.3.4 api-foo.example.com -> upstream:8080 "GET / HTTP/1.1" 200 12ms cache_hit=0 waf_block=0`,
		`1.2.3.4 other.example.com -> upstream:8080 "GET / HTTP/1.1" 200 12ms cache_hit=0 waf_block=0`,
	}
	out := FilterAccessLinesForRoute(cfg, 0, -1, lines)
	if len(out) != 1 {
		t.Fatalf("got %d lines, want 1", len(out))
	}
}
