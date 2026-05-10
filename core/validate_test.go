package core

import (
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestValidateConfig_RedirectWithoutService(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "redirect-only.example.com",
				Backend: rule.Backend{
					Redirect: rule.Redirect{
						URL: "https://www.example.com/",
					},
				},
			},
			{
				Host: "redirect-with-path.example.com",
				Backend: rule.Backend{
					Redirect: rule.Redirect{
						URL: "https://fallback.example.com/",
					},
				},
				Paths: []rule.Path{
					{
						Path: "^/api/",
						Backend: rule.Backend{
							Service: service.Service{
								Name:     "api-svc",
								Port:     8080,
								Protocol: "http",
							},
						},
					},
					{
						Path: "^/go$",
						Backend: rule.Backend{
							Redirect: rule.Redirect{
								URL: "https://else.example.com/",
							},
						},
					},
				},
			},
		},
	}

	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected redirect-only backends to validate, got: %v", err)
	}
}

func TestValidateConfig_ServiceRequiredWhenNoRedirect(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host:    "broken.example.com",
				Backend: rule.Backend{},
			},
		},
	}

	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected error when backend has neither redirect nor service name")
	}
}

func TestValidateConfig_RedirectWithNamedServiceRejected(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "dual.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "upstream-svc",
						Port:     8080,
						Protocol: "http",
					},
					Redirect: rule.Redirect{
						URL: "https://fallback.example.com/",
					},
				},
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error when backend.redirect and backend.service are both set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
	if !strings.Contains(err.Error(), `path="/"`) {
		t.Fatalf("expected rule-level backend path=/ in error, got: %v", err)
	}
}

func TestValidateConfig_RedirectWithNamedServiceRejected_ShowsPathPattern(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "api.example.com",
				Backend: rule.Backend{
					Redirect: rule.Redirect{
						URL: "https://fallback.example.com/",
					},
				},
				Paths: []rule.Path{
					{
						Path: "^/api/v1/",
						Backend: rule.Backend{
							Service: service.Service{
								Name:     "upstream-svc",
								Port:     8080,
								Protocol: "http",
							},
							Redirect: rule.Redirect{
								URL: "https://else.example.com/",
							},
						},
					},
				},
			},
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for path-level mutually exclusive backend")
	}
	if !strings.Contains(err.Error(), `path="^/api/v1/"`) {
		t.Fatalf("expected configured path pattern in error, got: %v", err)
	}
}
