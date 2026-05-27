package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-zoox/jwt"
)

func TestValidateJWTAuth_HS256(t *testing.T) {
	secret := "test-jwt-secret"
	j := jwt.New(secret, &jwt.Options{
		Algorithm: jwt.AlgHS256,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	token, err := j.Sign(map[string]interface{}{"sub": "user-1"})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	s := &Service{
		Auth: Auth{
			Type: "jwt",
			JWT: JWTAuth{
				Secret: secret,
			},
		},
	}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err := s.validateJWTAuth(req); err != nil {
		t.Fatalf("expected valid jwt, got %v", err)
	}
}

func TestValidateJWTAuth_LegacySecretField(t *testing.T) {
	secret := "legacy-secret"
	j := jwt.New(secret)
	token, err := j.Sign(map[string]interface{}{"ok": true})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	s := &Service{
		Auth: Auth{
			Type:   "jwt",
			Secret: secret,
		},
	}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err := s.validateJWTAuth(req); err != nil {
		t.Fatalf("expected valid jwt via auth.secret, got %v", err)
	}
}

func TestValidateJWTAuth_InvalidToken(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "jwt",
			JWT:  JWTAuth{Secret: "secret"},
		},
	}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	if err := s.validateJWTAuth(req); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateJWTAuth_MissingBearer(t *testing.T) {
	s := &Service{Auth: Auth{Type: "jwt", JWT: JWTAuth{Secret: "s"}}}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	if err := s.validateJWTAuth(req); err == nil {
		t.Fatal("expected missing bearer error")
	}
}

func TestValidateJWTAuth_WrongSecret(t *testing.T) {
	token, err := jwt.New("right-secret").Sign(map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Auth: Auth{Type: "jwt", JWT: JWTAuth{Secret: "wrong-secret"}}}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err := s.validateJWTAuth(req); err == nil {
		t.Fatal("expected invalid jwt")
	}
}

func TestValidateAuth_JWTType(t *testing.T) {
	token, err := jwt.New("secret").Sign(map[string]any{"sub": "u"})
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Auth: Auth{Type: "jwt", JWT: JWTAuth{Secret: "secret"}}}
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err := s.ValidateAuth(req); err != nil {
		t.Fatalf("ValidateAuth jwt: %v", err)
	}
}

func TestEnsureOpenIDScopes(t *testing.T) {
	got := ensureOpenIDScopes([]string{"profile", "email"})
	if got[0] != "openid" {
		t.Fatalf("expected openid injected, got %v", got)
	}
	got2 := ensureOpenIDScopes([]string{"openid", "email"})
	if len(got2) != 2 || got2[0] != "openid" {
		t.Fatalf("unexpected scopes: %v", got2)
	}
}
