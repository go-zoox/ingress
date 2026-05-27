package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-zoox/jwt"
)

func (s *Service) validateJWTAuth(req *http.Request) error {
	cfg := s.effectiveJWTConfig()
	if cfg.Secret == "" && cfg.PublicKey == "" {
		return fmt.Errorf("jwt secret or public_key is required")
	}

	token, err := bearerTokenFromRequest(req)
	if err != nil {
		return err
	}

	secret := cfg.Secret
	if secret == "" {
		secret = cfg.PublicKey
	}

	var verifyOpts *jwt.VerifyOptions
	if cfg.Issuer != "" || cfg.Audience != "" {
		verifyOpts = &jwt.VerifyOptions{
			Issuer:   cfg.Issuer,
			Audience: cfg.Audience,
		}
	}

	_, _, err = jwt.Verify(secret, token, verifyOpts)
	if err != nil {
		return fmt.Errorf("invalid jwt: %w", err)
	}
	return nil
}

func (s *Service) effectiveJWTConfig() JWTAuth {
	cfg := s.Auth.JWT
	if cfg.Secret == "" && s.Auth.Secret != "" {
		cfg.Secret = s.Auth.Secret
	}
	if strings.TrimSpace(cfg.Algorithm) == "" {
		cfg.Algorithm = "HS256"
	}
	return cfg
}

func bearerTokenFromRequest(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization scheme, expected Bearer")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return "", fmt.Errorf("bearer token missing")
	}
	return token, nil
}
