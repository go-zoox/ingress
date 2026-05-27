package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-zoox/jwt"
	"github.com/go-zoox/zoox"
)

// ValidateOIDCSession runs redirect-based OIDC (provider + client credentials).
func (s *Service) ValidateOIDCSession(ctx *zoox.Context) (redirected bool, err error) {
	cfg := s.Auth.OIDC
	if strings.TrimSpace(cfg.Provider) == "" {
		return false, fmt.Errorf("oidc provider is required for session flow")
	}
	oauthCfg := OAuth2Auth{
		Provider:     cfg.Provider,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       ensureOpenIDScopes(cfg.Scopes),
	}
	clone := *s
	clone.Auth.OAuth2 = oauthCfg
	return clone.ValidateOAuth2(ctx)
}

func ensureOpenIDScopes(scopes []string) []string {
	for _, s := range scopes {
		if strings.EqualFold(strings.TrimSpace(s), "openid") {
			return scopes
		}
	}
	out := make([]string, 0, len(scopes)+1)
	out = append(out, "openid")
	out = append(out, scopes...)
	return out
}

func (s *Service) validateOIDCBearer(req *http.Request) error {
	cfg := s.Auth.OIDC
	issuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	if issuer == "" {
		return fmt.Errorf("oidc issuer is required for bearer validation")
	}

	token, err := bearerTokenFromRequest(req)
	if err != nil {
		return err
	}

	header, _, headerRaw, _, _, err := jwt.Parse(token)
	if err != nil {
		return fmt.Errorf("invalid oidc token: %w", err)
	}

	kid := jwtKeyIDFromRaw(headerRaw)
	_ = header // algorithm available if needed later

	key, err := oidcVerifierForIssuer(issuer).publicKey(req.Context(), kid)
	if err != nil {
		return fmt.Errorf("oidc jwks: %w", err)
	}

	var verifyOpts *jwt.VerifyOptions
	if cfg.Audience != "" || issuer != "" {
		verifyOpts = &jwt.VerifyOptions{
			Issuer:   issuer,
			Audience: cfg.Audience,
		}
	}

	_, _, err = jwt.Verify(key, token, verifyOpts)
	if err != nil {
		return fmt.Errorf("invalid oidc token: %w", err)
	}
	return nil
}

type oidcVerifier struct {
	issuer string
	mu     sync.RWMutex
	jwks   map[string]string
	expiry time.Time
}

var oidcVerifiers sync.Map

func oidcVerifierForIssuer(issuer string) *oidcVerifier {
	if v, ok := oidcVerifiers.Load(issuer); ok {
		return v.(*oidcVerifier)
	}
	v := &oidcVerifier{issuer: issuer, jwks: map[string]string{}}
	actual, _ := oidcVerifiers.LoadOrStore(issuer, v)
	return actual.(*oidcVerifier)
}

func (v *oidcVerifier) publicKey(ctx context.Context, kid string) (string, error) {
	v.mu.RLock()
	if time.Now().Before(v.expiry) {
		if pemKey, ok := lookupJWKS(v.jwks, kid); ok {
			v.mu.RUnlock()
			return pemKey, nil
		}
	}
	v.mu.RUnlock()

	if err := v.refresh(ctx); err != nil {
		return "", err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	if pemKey, ok := lookupJWKS(v.jwks, kid); ok {
		return pemKey, nil
	}
	return "", fmt.Errorf("jwks key %q not found", kid)
}

func lookupJWKS(keys map[string]string, kid string) (string, bool) {
	if kid != "" {
		if pem, ok := keys[kid]; ok {
			return pem, true
		}
	}
	for _, pem := range keys {
		return pem, true
	}
	return "", false
}

func (v *oidcVerifier) refresh(ctx context.Context) error {
	discoveryURL := v.issuer + "/.well-known/openid-configuration"
	body, err := httpGet(ctx, discoveryURL)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	var disc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &disc); err != nil {
		return fmt.Errorf("discovery json: %w", err)
	}
	if strings.TrimSpace(disc.JWKSURI) == "" {
		return fmt.Errorf("discovery missing jwks_uri")
	}

	jwksBody, err := httpGet(ctx, disc.JWKSURI)
	if err != nil {
		return fmt.Errorf("jwks: %w", err)
	}
	var jwks struct {
		Keys []jwkKey `json:"keys"`
	}
	if err := json.Unmarshal(jwksBody, &jwks); err != nil {
		return fmt.Errorf("jwks json: %w", err)
	}

	keys := map[string]string{}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		pemKey, err := rsaJWKToPEM(k.N, k.E)
		if err != nil {
			continue
		}
		kid := k.Kid
		if kid == "" {
			kid = "_default"
		}
		keys[kid] = pemKey
	}
	if len(keys) == 0 {
		return fmt.Errorf("no usable RSA signing keys in jwks")
	}

	v.mu.Lock()
	v.jwks = keys
	v.expiry = time.Now().Add(time.Hour)
	v.mu.Unlock()
	return nil
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func jwtKeyIDFromRaw(headerRaw string) string {
	if headerRaw == "" {
		return ""
	}
	b, err := base64.RawURLEncoding.DecodeString(headerRaw)
	if err != nil {
		return ""
	}
	var h struct {
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return ""
	}
	return h.Kid
}

func rsaJWKToPEM(nB64, eB64 string) (string, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return "", err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return "", err
	}
	eInt := 0
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		eInt = 65537
	}
	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}
