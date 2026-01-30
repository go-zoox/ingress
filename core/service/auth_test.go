package service

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestValidateAuth_NoAuth(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "",
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	err := s.ValidateAuth(req)
	if err != nil {
		t.Fatalf("expected no error when auth is not configured, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_Success_SingleUser(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("admin:admin123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	err := s.ValidateAuth(req)
	if err != nil {
		t.Fatalf("expected authentication to succeed, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_Success_MultipleUsers(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
					{
						Username: "user1",
						Password: "user123",
					},
					{
						Username: "user2",
						Password: "user456",
					},
				},
			},
		},
	}

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{"first user", "admin", "admin123"},
		{"second user", "user1", "user123"},
		{"third user", "user2", "user456"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			credentials := base64.StdEncoding.EncodeToString([]byte(tc.username + ":" + tc.password))
			req.Header.Set("Authorization", "Basic "+credentials)

			err := s.ValidateAuth(req)
			if err != nil {
				t.Fatalf("expected authentication to succeed for user %s, got: %v", tc.username, err)
			}
		})
	}
}

func TestValidateAuth_BasicAuth_MissingHeader(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when Authorization header is missing")
	}
	if err.Error() != "authorization header missing" {
		t.Fatalf("expected 'authorization header missing' error, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_InvalidScheme(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token123")

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when authorization scheme is invalid")
	}
	if err.Error() != "invalid authorization scheme, expected Basic" {
		t.Fatalf("expected 'invalid authorization scheme, expected Basic' error, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_InvalidBase64(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic invalid-base64!!!")

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when base64 encoding is invalid")
	}
	if err.Error() == "authorization header missing" {
		t.Fatalf("expected base64 decoding error, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_InvalidFormat(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	// Missing colon separator
	credentials := base64.StdEncoding.EncodeToString([]byte("adminadmin123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when credentials format is invalid")
	}
	if err.Error() != "invalid credentials format" {
		t.Fatalf("expected 'invalid credentials format' error, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_InvalidCredentials(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "admin123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("wrong:password"))
	req.Header.Set("Authorization", "Basic "+credentials)

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when credentials are invalid")
	}
	if err.Error() != "invalid credentials" {
		t.Fatalf("expected 'invalid credentials' error, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_NoUsersConfigured(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("admin:admin123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when no users are configured")
	}
	if err.Error() != "no basic auth users configured" {
		t.Fatalf("expected 'no basic auth users configured' error, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_Success_SingleToken(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{"token123"},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token123")

	err := s.ValidateAuth(req)
	if err != nil {
		t.Fatalf("expected authentication to succeed, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_Success_MultipleTokens(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{
					"token1-abc123xyz",
					"token2-def456uvw",
					"token3-ghi789rst",
				},
			},
		},
	}

	testCases := []struct {
		name  string
		token string
	}{
		{"first token", "token1-abc123xyz"},
		{"second token", "token2-def456uvw"},
		{"third token", "token3-ghi789rst"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)

			err := s.ValidateAuth(req)
			if err != nil {
				t.Fatalf("expected authentication to succeed for token %s, got: %v", tc.token, err)
			}
		})
	}
}

func TestValidateAuth_BearerToken_WithSpaces(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{"token123"},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer  token123  ")

	err := s.ValidateAuth(req)
	if err != nil {
		t.Fatalf("expected authentication to succeed with spaces, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_MissingHeader(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{"token123"},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when Authorization header is missing")
	}
	if err.Error() != "authorization header missing" {
		t.Fatalf("expected 'authorization header missing' error, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_InvalidScheme(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{"token123"},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dG9rZW4xMjM=")

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when authorization scheme is invalid")
	}
	if err.Error() != "invalid authorization scheme, expected Bearer" {
		t.Fatalf("expected 'invalid authorization scheme, expected Bearer' error, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_InvalidToken(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{"token123"},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when token is invalid")
	}
	if err.Error() != "invalid token" {
		t.Fatalf("expected 'invalid token' error, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_NoTokensConfigured(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer token123")

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when no tokens are configured")
	}
	if err.Error() != "no bearer tokens configured" {
		t.Fatalf("expected 'no bearer tokens configured' error, got: %v", err)
	}
}

func TestValidateAuth_UnsupportedType(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "oauth2",
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when auth type is unsupported")
	}
	if err.Error() != "unsupported auth type: oauth2" {
		t.Fatalf("expected 'unsupported auth type: oauth2' error, got: %v", err)
	}
}

func TestValidateAuth_BasicAuth_PasswordWithColon(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "basic",
			Basic: BasicAuth{
				Users: []BasicUser{
					{
						Username: "admin",
						Password: "pass:word:123",
					},
				},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("admin:pass:word:123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	err := s.ValidateAuth(req)
	if err != nil {
		t.Fatalf("expected authentication to succeed with password containing colon, got: %v", err)
	}
}

func TestValidateAuth_BearerToken_EmptyToken(t *testing.T) {
	s := &Service{
		Auth: Auth{
			Type: "bearer",
			Bearer: BearerAuth{
				Tokens: []string{"token123"},
			},
		},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ")

	err := s.ValidateAuth(req)
	if err == nil {
		t.Fatal("expected error when token is empty")
	}
	if err.Error() != "invalid token" {
		t.Fatalf("expected 'invalid token' error, got: %v", err)
	}
}
