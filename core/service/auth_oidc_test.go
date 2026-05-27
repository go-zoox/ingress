package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-zoox/jwt"
)

func TestJwtKeyIDFromRaw(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"key-1","typ":"JWT"}`))
	if got := jwtKeyIDFromRaw(header); got != "key-1" {
		t.Fatalf("kid=%q", got)
	}
}

func TestRsaJWKToPEM_roundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())

	pemKey, err := rsaJWKToPEM(n, e)
	if err != nil {
		t.Fatalf("rsaJWKToPEM: %v", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	token, err := jwt.Sign(string(privPEM), map[string]any{"sub": "oidc-user"}, &jwt.SignOptions{
		Algorithm: jwt.AlgRS256,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, _, err = jwt.Verify(pemKey, token, nil); err != nil {
		t.Fatalf("Verify with JWK-derived PEM: %v", err)
	}
}

func TestValidateOIDCBearer_MockIssuer(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-rsa"
	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())

	var issuer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/.well-known/openid-configuration"):
			_ = json.NewEncoder(w).Encode(map[string]string{
				"jwks_uri": issuer + "/jwks",
			})
		case strings.HasSuffix(r.URL.Path, "/jwks"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"n":   n,
					"e":   e,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	issuer = strings.TrimRight(srv.URL, "/")

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	token, err := jwt.Sign(string(privPEM), map[string]any{"sub": "api-user"}, &jwt.SignOptions{
		Algorithm: jwt.AlgRS256,
		Issuer:    issuer,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	s := &Service{
		Auth: Auth{
			Type: "oidc",
			OIDC: OIDCAuth{Issuer: issuer},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err := s.validateOIDCBearer(req); err != nil {
		t.Fatalf("validateOIDCBearer: %v", err)
	}
}

func TestValidateAuth_OIDCProviderSessionSkipsBearer(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "oidc",
			OIDC: OIDCAuth{Provider: "google"},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err := s.ValidateAuth(req); err != nil {
		t.Fatalf("session oidc should pass ValidateAuth: %v", err)
	}
}

func TestValidateOIDCBearer_MissingIssuer(t *testing.T) {
	s := &Service{Auth: Auth{Type: "oidc", OIDC: OIDCAuth{}}}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Authorization", "Bearer x")
	if err := s.validateOIDCBearer(req); err == nil {
		t.Fatal("expected error without issuer")
	}
}
