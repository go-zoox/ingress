package service

import (
	"encoding/json"
	"testing"
	"time"

	gozooxoauth2 "github.com/go-zoox/oauth2"
)

func TestSupportedOAuth2Providers(t *testing.T) {
	expected := []string{
		"doreamon", "github", "feishu", "gitlab", "slack",
		"kakao", "google", "microsoft", "auth0", "okta",
	}
	for _, p := range expected {
		if !supportedOAuth2Providers[p] {
			t.Fatalf("expected provider %q to be supported", p)
		}
	}
	if supportedOAuth2Providers["unsupported"] {
		t.Fatal("expected 'unsupported' provider to not be in the list")
	}
}

func TestBuildConnectJWT_HS256(t *testing.T) {
	s := &Service{
		Auth: Auth{
			OAuth2: OAuth2Auth{
				Connect: OAuth2Connect{
					Enabled: true,
					JWT: OAuth2ConnectJWT{
						Secret:    "my-secret-key",
						Algorithm: "hs256",
						ExpiresIn: "5m",
					},
				},
			},
		},
	}

	user := &gozooxoauth2.User{
		ID:       "12345",
		Username: "testuser",
		Email:    "test@example.com",
		Nickname: "Test User",
		Avatar:   "https://example.com/avatar.png",
	}

	token, err := s.BuildConnectJWT(user)
	if err != nil {
		t.Fatalf("BuildConnectJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty JWT token")
	}
}

func TestBuildConnectJWT_DefaultAlgorithm(t *testing.T) {
	s := &Service{
		Auth: Auth{
			OAuth2: OAuth2Auth{
				Connect: OAuth2Connect{
					Enabled: true,
					JWT: OAuth2ConnectJWT{
						Secret:    "my-secret-key",
						ExpiresIn: "1h",
					},
				},
			},
		},
	}

	user := &gozooxoauth2.User{
		ID:    "abc",
		Email: "test@example.com",
	}

	token, err := s.BuildConnectJWT(user)
	if err != nil {
		t.Fatalf("BuildConnectJWT with defaults failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty JWT")
	}
}

func TestBuildConnectJWT_DefaultExpiresIn(t *testing.T) {
	s := &Service{
		Auth: Auth{
			OAuth2: OAuth2Auth{
				Connect: OAuth2Connect{
					Enabled: true,
					JWT: OAuth2ConnectJWT{
						Secret: "my-secret-key",
					},
				},
			},
		},
	}

	user := &gozooxoauth2.User{
		ID:       "1",
		Username: "u",
	}

	token, err := s.BuildConnectJWT(user)
	if err != nil {
		t.Fatalf("BuildConnectJWT default expires_in failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty JWT")
	}
}

func TestBuildConnectJWT_UserSerialization(t *testing.T) {
	s := &Service{
		Auth: Auth{
			OAuth2: OAuth2Auth{
				Connect: OAuth2Connect{
					Enabled: true,
					JWT: OAuth2ConnectJWT{
						Secret:    "test-secret",
						Algorithm: "hs256",
						ExpiresIn: "10m",
					},
				},
			},
		},
	}

	user := &gozooxoauth2.User{
		ID:       "user-001",
		Username: "johndoe",
		Email:    "john@example.com",
		Nickname: "John",
		Avatar:   "https://img.example.com/john.png",
		Groups:   []string{"admin", "developer"},
	}

	// Serialize user to JSON and back to simulate session round-trip.
	userJSON, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}

	var restored gozooxoauth2.User
	if err := json.Unmarshal(userJSON, &restored); err != nil {
		t.Fatalf("unmarshal user: %v", err)
	}

	token, err := s.BuildConnectJWT(&restored)
	if err != nil {
		t.Fatalf("BuildConnectJWT after round-trip failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty JWT")
	}
}

func TestBuildConnectJWT_InvalidExpiresIn(t *testing.T) {
	s := &Service{
		Auth: Auth{
			OAuth2: OAuth2Auth{
				Connect: OAuth2Connect{
					Enabled: true,
					JWT: OAuth2ConnectJWT{
						Secret:    "test-secret",
						ExpiresIn: "invalid",
					},
				},
			},
		},
	}

	user := &gozooxoauth2.User{ID: "1"}
	_, err := s.BuildConnectJWT(user)
	if err == nil {
		t.Fatal("expected error for invalid expires_in")
	}
}

func TestGenerateRandomState(t *testing.T) {
	state1, err := generateRandomState()
	if err != nil {
		t.Fatalf("generateRandomState failed: %v", err)
	}
	if len(state1) != 32 {
		t.Fatalf("expected 32 hex chars (16 bytes), got %d", len(state1))
	}

	state2, err := generateRandomState()
	if err != nil {
		t.Fatalf("generateRandomState failed: %v", err)
	}
	if state1 == state2 {
		t.Fatal("expected different random states")
	}
}

func TestOAuth2ConnectJWT_ExpiresIn(t *testing.T) {
	s := &Service{
		Auth: Auth{
			OAuth2: OAuth2Auth{
				Connect: OAuth2Connect{
					Enabled: true,
					JWT: OAuth2ConnectJWT{
						Secret:    "secret",
						ExpiresIn: "2s",
					},
				},
			},
		},
	}

	user := &gozooxoauth2.User{ID: "x"}

	start := time.Now()
	token, err := s.BuildConnectJWT(user)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("BuildConnectJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty JWT")
	}
	// ExpiresIn parsing should work (2s is very short but valid).
	if elapsed > time.Second {
		t.Logf("BuildConnectJWT took %v (may be fine for CI)", elapsed)
	}
}

// TestDefaultScopesForProvider verifies that each provider with configured
// defaults returns the expected scopes when none are specified in config.
func TestDefaultScopesForProvider(t *testing.T) {
	tests := []struct {
		provider string
		expected []string
	}{
		{"github", []string{"user:email"}},
		{"gitlab", []string{"read_user"}},
		{"google", []string{"openid", "profile", "email"}},
		{"microsoft", []string{"openid", "profile", "email"}},
		{"feishu", []string{"user:email"}},
		{"slack", []string{"users:read"}},
		{"auth0", []string{"openid", "profile", "email"}},
		{"okta", []string{"openid", "profile", "email"}},
	}

	for _, tc := range tests {
		t.Run(tc.provider, func(t *testing.T) {
			scopes, ok := defaultOAuth2Scopes[tc.provider]
			if !ok {
				t.Fatalf("expected default scopes for %s, got none", tc.provider)
			}
			if len(scopes) != len(tc.expected) {
				t.Fatalf("expected %d scopes, got %d: %v", len(tc.expected), len(scopes), scopes)
			}
			for i, s := range scopes {
				if s != tc.expected[i] {
					t.Fatalf("scope[%d]: expected %q, got %q", i, tc.expected[i], s)
				}
			}
		})
	}
}

// TestNoDefaultScopesProvider verifies that providers without explicit defaults
// (doreamon, kakao) return nil/empty, meaning they rely on provider defaults.
func TestNoDefaultScopesProvider(t *testing.T) {
	for _, p := range []string{"doreamon", "kakao"} {
		t.Run(p, func(t *testing.T) {
			scopes, ok := defaultOAuth2Scopes[p]
			if ok {
				t.Fatalf("%s: expected no default scopes, got %v", p, scopes)
			}
		})
	}
}
