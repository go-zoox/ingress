package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfigFile(t *testing.T) {
	t.Run("valid yaml", func(t *testing.T) {
		dir := t.TempDir()
		configFilePath := filepath.Join(dir, "config.yaml")
		content := `
port: 8080
rules:
  - host: example.com
    backend:
      service:
        name: backend-svc
`

		if err := os.WriteFile(configFilePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		if err := validateConfigFile(configFilePath); err != nil {
			t.Fatalf("expected valid config, got error: %v", err)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		configFilePath := filepath.Join(dir, "config.yaml")
		content := `
port: 8080
rules:
  - host: example.com
    backend:
      service:
        name: backend-svc
    broken: [1,2
`

		if err := os.WriteFile(configFilePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		err := validateConfigFile(configFilePath)
		if err == nil {
			t.Fatal("expected invalid yaml error, got nil")
		}
		if !strings.Contains(err.Error(), "yaml syntax error") {
			t.Fatalf("expected yaml syntax error message, got: %v", err)
		}
	})

	t.Run("unsupported configuration", func(t *testing.T) {
		dir := t.TempDir()
		configFilePath := filepath.Join(dir, "config.yaml")
		content := `
rules:
  - host: example.com
    backend:
      type: invalid_backend_type
`

		if err := os.WriteFile(configFilePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		err := validateConfigFile(configFilePath)
		if err == nil {
			t.Fatal("expected unsupported configuration error, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported configuration") {
			t.Fatalf("expected unsupported configuration message, got: %v", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		configFilePath := filepath.Join(dir, "missing.yaml")

		if err := validateConfigFile(configFilePath); err == nil {
			t.Fatal("expected missing file error, got nil")
		}
	})
}
