package core

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/ratelimit"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func validateRateLimit(cfg rule.RateLimit, loc string) error {
	if cfg.Enabled != nil && !*cfg.Enabled {
		return nil
	}
	if cfg.Requests <= 0 && cfg.Enabled == nil {
		return nil
	}
	if cfg.Requests <= 0 {
		return fmt.Errorf("%s: rate_limit.requests must be positive when enabled", loc)
	}
	if cfg.Period <= 0 {
		return fmt.Errorf("%s: rate_limit.period must be positive", loc)
	}
	key := strings.ToLower(strings.TrimSpace(cfg.Key))
	if key == "" {
		key = ratelimit.KeyIP
	}
	switch key {
	case ratelimit.KeyGlobal, ratelimit.KeyRoute, ratelimit.KeyIP, ratelimit.KeyHeader:
	default:
		return fmt.Errorf("%s: rate_limit.key must be global, route, ip, or header", loc)
	}
	if key == ratelimit.KeyHeader && strings.TrimSpace(cfg.Header) == "" {
		return fmt.Errorf("%s: rate_limit.header is required when key is header", loc)
	}
	return nil
}

func validateServiceAuth(auth service.Auth, loc string) error {
	if !auth.IsEnabled() {
		return nil
	}
	authType := strings.ToLower(strings.TrimSpace(auth.Type))
	switch authType {
	case "", "basic", "bearer", "oauth2", "jwt", "oidc":
	default:
		return fmt.Errorf("%s: unsupported auth.type %q", loc, auth.Type)
	}
	if authType == "" {
		return nil
	}

	switch authType {
	case "basic":
		if len(auth.Basic.Users) == 0 {
			return fmt.Errorf("%s: auth.basic.users is required", loc)
		}
	case "bearer":
		if len(auth.Bearer.Tokens) == 0 {
			return fmt.Errorf("%s: auth.bearer.tokens is required", loc)
		}
	case "jwt":
		cfg := auth.JWT
		secret := cfg.Secret
		if secret == "" {
			secret = auth.Secret
		}
		if secret == "" && strings.TrimSpace(cfg.PublicKey) == "" {
			return fmt.Errorf("%s: auth.jwt.secret, auth.secret, or auth.jwt.public_key is required", loc)
		}
	case "oauth2":
		if strings.TrimSpace(auth.OAuth2.Provider) == "" {
			return fmt.Errorf("%s: auth.oauth2.provider is required", loc)
		}
		if strings.TrimSpace(auth.OAuth2.ClientID) == "" {
			return fmt.Errorf("%s: auth.oauth2.client_id is required", loc)
		}
		if strings.TrimSpace(auth.OAuth2.ClientSecret) == "" {
			return fmt.Errorf("%s: auth.oauth2.client_secret is required", loc)
		}
	case "oidc":
		hasProvider := strings.TrimSpace(auth.OIDC.Provider) != ""
		hasIssuer := strings.TrimSpace(auth.OIDC.Issuer) != ""
		if !hasProvider && !hasIssuer {
			return fmt.Errorf("%s: auth.oidc.provider or auth.oidc.issuer is required", loc)
		}
		if hasProvider {
			if strings.TrimSpace(auth.OIDC.ClientID) == "" {
				return fmt.Errorf("%s: auth.oidc.client_id is required when provider is set", loc)
			}
			if strings.TrimSpace(auth.OIDC.ClientSecret) == "" {
				return fmt.Errorf("%s: auth.oidc.client_secret is required when provider is set", loc)
			}
		}
	}
	return nil
}
