package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths_defaultLogPaths(t *testing.T) {
	root := t.TempDir()
	adminDir := filepath.Join(root, "admin")
	exampleDir := filepath.Join(root, "examples", "admin-console")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ingressFile := filepath.Join(exampleDir, "ingress.yaml")
	if err := os.WriteFile(ingressFile, []byte(`version: v1
logging:
  enable: true
  level: warn
`), 0o644); err != nil {
		t.Fatal(err)
	}
	adminFile := filepath.Join(adminDir, "admin.yaml")
	if err := os.MkdirAll(adminDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Ingress: Ingress{
			ConfigPath: filepath.Join("..", "examples", "admin-console", "ingress.yaml"),
		},
	}
	if err := ResolvePaths(&cfg, adminFile); err != nil {
		t.Fatal(err)
	}
	if cfg.Ingress.LogPath != "/var/log/ingress/access.log" {
		t.Fatalf("log_path: got %q", cfg.Ingress.LogPath)
	}
	if cfg.Ingress.ErrorLogPath != "/var/log/ingress/error.log" {
		t.Fatalf("error_log_path: got %q", cfg.Ingress.ErrorLogPath)
	}
}
