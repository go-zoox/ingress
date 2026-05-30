package config

import (
	"fmt"
	"strings"
)

// Auth configures Admin Console authentication.
type Auth struct {
	Type  string
	Basic AuthBasic
	OAuth AuthOAuth
}

type AuthBasic struct {
	Username string
	Password string
}

type AuthOAuth struct {
	Provider     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// EffectiveAuthType returns none (default), basic, or oauth.
func EffectiveAuthType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "none", "basic", "oauth":
		return strings.ToLower(strings.TrimSpace(t))
	default:
		return "none"
	}
}

func (a *Auth) Validate() error {
	if a == nil {
		return nil
	}
	switch EffectiveAuthType(a.Type) {
	case "none", "basic":
		return nil
	case "oauth":
		if strings.TrimSpace(a.OAuth.Provider) == "" {
			return fmt.Errorf("admin.auth.oauth.provider is required when auth.type is oauth")
		}
		if strings.TrimSpace(a.OAuth.ClientID) == "" {
			return fmt.Errorf("admin.auth.oauth.client_id is required when auth.type is oauth")
		}
		if strings.TrimSpace(a.OAuth.ClientSecret) == "" {
			return fmt.Errorf("admin.auth.oauth.client_secret is required when auth.type is oauth")
		}
		return nil
	default:
		return fmt.Errorf("admin.auth.type must be none, basic, or oauth")
	}
}
