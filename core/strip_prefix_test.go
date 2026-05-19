package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestStripPrefixRewriteRule_literalPath(t *testing.T) {
	rw, err := stripPrefixRewriteRule("/api/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if rw != "^/api/dashboard/?(.*):/$1" {
		t.Fatalf("got %q", rw)
	}
}

func TestStripPrefixRewriteRule_captureSuffix(t *testing.T) {
	rw, err := stripPrefixRewriteRule("/httpbin.org/(.*)")
	if err != nil {
		t.Fatal(err)
	}
	if rw != "^/httpbin.org/(.*):/$1" {
		t.Fatalf("got %q", rw)
	}
}

func TestApplyStripPrefix_expandsPathBackend(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "example.com",
				Paths: []rule.Path{
					{
						Path: "/api/dashboard",
						Backend: rule.Backend{
							Service: service.Service{
								Name:        "upstream.example.com",
								StripPrefix: true,
							},
						},
					},
				},
			},
		},
	}
	if err := inferBackendTypes(cfg); err != nil {
		t.Fatal(err)
	}
	svc := cfg.Rules[0].Paths[0].Backend.Service
	if svc.StripPrefix {
		t.Fatal("expected strip_prefix cleared after expand")
	}
	if len(svc.Request.Path.Rewrites) != 1 || svc.Request.Path.Rewrites[0] != "^/api/dashboard/?(.*):/$1" {
		t.Fatalf("rewrites: %#v", svc.Request.Path.Rewrites)
	}
}

func TestApplyStripPrefix_rejectsRuleLevel(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:        "upstream.example.com",
						StripPrefix: true,
					},
				},
			},
		},
	}
	err := inferBackendTypes(cfg)
	if err == nil {
		t.Fatal("expected error for rule-level strip_prefix")
	}
}

func TestApplyStripPrefix_rejectsWithRewrites(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "example.com",
				Paths: []rule.Path{
					{
						Path: "/api",
						Backend: rule.Backend{
							Service: service.Service{
								Name:        "upstream.example.com",
								StripPrefix: true,
								Request: service.Request{
									Path: service.RequestPath{
										Rewrites: []string{"^/api:/"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err := inferBackendTypes(cfg)
	if err == nil {
		t.Fatal("expected error when strip_prefix and rewrites are both set")
	}
}
