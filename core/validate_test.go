package core

import (
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/ingress/core/waf"
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
		t.Fatal("expected error when backend.redirect and backend.service are both set without explicit type")
	}
	if !strings.Contains(err.Error(), "ambiguous backend") {
		t.Fatalf("expected ambiguous backend error, got: %v", err)
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
		t.Fatal("expected error for path-level ambiguous backend")
	}
	if !strings.Contains(err.Error(), "ambiguous backend") {
		t.Fatalf("expected ambiguous backend error, got: %v", err)
	}
	if !strings.Contains(err.Error(), `path="^/api/v1/"`) {
		t.Fatalf("expected configured path pattern in error, got: %v", err)
	}
}

func TestValidateConfig_ExplicitServiceWithRedirectRejected(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "x.example.com",
				Backend: rule.Backend{
					Type: backendTypeService,
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
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error when type service combined with redirect block")
	}
	if !strings.Contains(err.Error(), `backend.type is "service" but backend.redirect`) {
		t.Fatalf("expected service+redirect conflict error, got: %v", err)
	}
}

func TestValidateConfig_InfersRedirectType(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "r.example.com",
				Backend: rule.Backend{
					Redirect: rule.Redirect{URL: "https://z.example/"},
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Rules[0].Backend.Type != backendTypeRedirect {
		t.Fatalf("expected inferred type redirect, got %q", cfg.Rules[0].Backend.Type)
	}
}

func TestValidateConfig_WAFInvalidCIDRError(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		WAF: rule.WAF{
			Enabled:        true,
			DisableBuiltin: true,
			Deny:           []string{"not-an-ip-address"},
			Rules:          []rule.WAFRule{},
		},
		Rules: []rule.Rule{
			{
				Host: "waf.example.com",
				Backend: rule.Backend{
					Service: service.Service{Name: "svc", Port: 8080, Protocol: "http"},
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err == nil || !strings.Contains(err.Error(), "waf") {
		t.Fatalf("expected waf deny error, got %v", err)
	}
}

func TestValidateConfig_WAFPasses(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		WAF: rule.WAF{
			Enabled:        true,
			DisableBuiltin: true,
			Rules: []rule.WAFRule{{
				ID:      "r1",
				Pattern: `BAD`,
				Type:    waf.PatternTypeContains,
				Targets: []string{waf.TargetPath},
			}},
		},
		Rules: []rule.Rule{
			{
				Host: "ok.example.com",
				Backend: rule.Backend{
					Service: service.Service{Name: "svc", Port: 8080, Protocol: "http"},
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfig_HandlerBackendCacheOK(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "h.example.com",
				Backend: rule.Backend{
					Type: backendTypeHandler,
					Cache: rule.BackendCache{
						Enabled: true,
					},
					Handler: rule.Handler{
						Type: handlerTypeStaticResponse,
						Body: "x",
					},
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfig_RedirectBackendCacheOK(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "r.example.com",
				Backend: rule.Backend{
					Type: backendTypeRedirect,
					Cache: rule.BackendCache{
						Enabled: true,
					},
					Redirect: rule.Redirect{
						URL: "https://example.com/",
					},
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfig_ServiceBackendCacheOK(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "ok.example.com",
				Backend: rule.Backend{
					Cache: rule.BackendCache{Enabled: true},
					Service: service.Service{
						Name:     "api",
						Port:     8080,
						Protocol: "http",
					},
				},
			},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfig_ServiceModeOnServiceOK(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "m.example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Name:     "api",
					Port:     8080,
					Protocol: "http",
					Mode:     backendModeExternal,
				},
			},
		}},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfig_ModeConflict(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "c.example.com",
			Backend: rule.Backend{
				Mode: backendModeInternal,
				Service: service.Service{
					Name:     "api",
					Port:     8080,
					Protocol: "http",
					Mode:     backendModeExternal,
				},
			},
		}},
	}
	if err := ValidateConfig(cfg); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}
