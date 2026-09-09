package core

import (
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
)

func secretKeyTestConfig() *Config {
	return &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "example.com",
				Backend: rule.Backend{
					Type:    backendTypeHandler,
					Handler: rule.Handler{Body: "ok"},
				},
			},
		},
	}
}

// TestApplySecretKey_FromConfig verifies a config `secret_key` is applied to the
// zoox app so OAuth2 session cookies can be decrypted consistently.
func TestApplySecretKey_FromConfig(t *testing.T) {
	cfg := secretKeyTestConfig()
	cfg.SecretKey = "config-secret"

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ins := c.(*core)

	if got := ins.app.Config.SecretKey; got != "config-secret" {
		t.Fatalf("expected app secret from config to be %q, got %q", "config-secret", got)
	}
}

// TestApplySecretKey_EnvOverridesConfig verifies SECRET_KEY env wins over config.
func TestApplySecretKey_EnvOverridesConfig(t *testing.T) {
	t.Setenv(zoox.BuiltInEnvSecretKey, "env-secret")
	cfg := secretKeyTestConfig()
	cfg.SecretKey = "config-secret"

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ins := c.(*core)

	if got := ins.app.Config.SecretKey; got != "env-secret" {
		t.Fatalf("expected SECRET_KEY env to override config, got %q", got)
	}
}

// TestApplySecretKey_EmptyByDefault verifies that when neither config nor env sets a
// key, app.Config.SecretKey stays empty so zoox applies its per-process default.
func TestApplySecretKey_EmptyByDefault(t *testing.T) {
	t.Setenv(zoox.BuiltInEnvSecretKey, "")
	cfg := secretKeyTestConfig()

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ins := c.(*core)

	if got := ins.app.Config.SecretKey; got != "" {
		t.Fatalf("expected empty app secret when unset, got %q", got)
	}
}
