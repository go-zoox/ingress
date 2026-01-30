package service

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// ValidateAuth validates the client's authentication credentials
// Returns error if authentication fails, nil if authentication succeeds
func (s *Service) ValidateAuth(req *http.Request) error {
	if s.Auth.Type == "" {
		return nil // No auth configured, allow all requests
	}

	switch s.Auth.Type {
	case "basic":
		return s.validateBasicAuth(req)
	case "bearer":
		return s.validateBearerAuth(req)
	default:
		return fmt.Errorf("unsupported auth type: %s", s.Auth.Type)
	}
}

// validateBasicAuth validates Basic Authentication from client request
func (s *Service) validateBasicAuth(req *http.Request) error {
	users := s.Auth.Basic.Users
	if len(users) == 0 {
		return fmt.Errorf("no basic auth users configured")
	}

	// Extract Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("authorization header missing")
	}

	// Check if it's Basic auth
	if !strings.HasPrefix(authHeader, "Basic ") {
		return fmt.Errorf("invalid authorization scheme, expected Basic")
	}

	// Decode credentials
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// Parse username:password
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid credentials format")
	}

	clientUsername := parts[0]
	clientPassword := parts[1]

	// Check against configured users
	for _, user := range users {
		if user.Username == clientUsername && user.Password == clientPassword {
			return nil // Authentication successful
		}
	}

	return fmt.Errorf("invalid credentials")
}

// validateBearerAuth validates Bearer Token from client request
func (s *Service) validateBearerAuth(req *http.Request) error {
	tokens := s.Auth.Bearer.Tokens
	if len(tokens) == 0 {
		return fmt.Errorf("no bearer tokens configured")
	}

	// Extract Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("authorization header missing")
	}

	// Check if it's Bearer auth
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("invalid authorization scheme, expected Bearer")
	}

	// Extract token
	clientToken := strings.TrimPrefix(authHeader, "Bearer ")
	clientToken = strings.TrimSpace(clientToken)

	// Check against configured tokens
	for _, token := range tokens {
		if token == clientToken {
			return nil // Authentication successful
		}
	}

	return fmt.Errorf("invalid token")
}
