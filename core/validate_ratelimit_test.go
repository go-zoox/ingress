package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestValidateConfig_RateLimit(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		RateLimit: rule.RateLimit{
			Requests: 10,
			Period:   60,
			Key:      "ip",
		},
		Rules: []rule.Rule{{
			Host: "example.com",
			Backend: rule.Backend{
				Service: service.Service{Name: "127.0.0.1", Port: 8081},
			},
		}},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateConfig_RateLimit_HeaderRequiresName(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "example.com",
			RateLimit: rule.RateLimit{
				Requests: 1,
				Period:   1,
				Key:      "header",
			},
			Backend: rule.Backend{
				Service: service.Service{Name: "127.0.0.1", Port: 8081},
			},
		}},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for missing header")
	}
}

func TestValidateConfig_JWTAuth(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Name: "127.0.0.1",
					Port: 8081,
					Auth: service.Auth{
						Type:   "jwt",
						Secret: "secret",
					},
				},
			},
		}},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected valid jwt auth config, got %v", err)
	}
}

func TestValidateConfig_RateLimit_InvalidKey(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		RateLimit: rule.RateLimit{
			Requests: 10,
			Period:   60,
			Key:      "bogus",
		},
		Rules: []rule.Rule{{
			Host: "example.com",
			Backend: rule.Backend{
				Service: service.Service{Name: "127.0.0.1", Port: 8081},
			},
		}},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestValidateConfig_JWTAuth_MissingSecret(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Name: "127.0.0.1",
					Port: 8081,
					Auth: service.Auth{Type: "jwt"},
				},
			},
		}},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected jwt secret required")
	}
}

func TestValidateConfig_OIDCAuth_ProviderRequiresClient(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Name: "127.0.0.1",
					Port: 8081,
					Auth: service.Auth{
						Type: "oidc",
						OIDC: service.OIDCAuth{Provider: "google"},
					},
				},
			},
		}},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected oidc client_id required")
	}
}

func TestValidateConfig_OIDCAuth_IssuerOnly(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{{
			Host: "example.com",
			Backend: rule.Backend{
				Service: service.Service{
					Name: "127.0.0.1",
					Port: 8081,
					Auth: service.Auth{
						Type: "oidc",
						OIDC: service.OIDCAuth{Issuer: "https://issuer.example.com"},
					},
				},
			},
		}},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("issuer-only oidc should validate: %v", err)
	}
}
