package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths_defaultLogPaths(t *testing.T) {
	root := t.TempDir()
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

	cfg := Config{
		IngressConfigPath: ingressFile,
	}
	if err := ResolvePaths(&cfg, ingressFile); err != nil {
		t.Fatal(err)
	}
	if cfg.AccessLogPath != "/var/log/ingress/access.log" {
		t.Fatalf("access_log_path: got %q", cfg.AccessLogPath)
	}
	if cfg.ErrorLogPath != "/var/log/ingress/error.log" {
		t.Fatalf("error_log_path: got %q", cfg.ErrorLogPath)
	}
}

func TestResolvePaths_relativeLoggingTransports(t *testing.T) {
	root := t.TempDir()
	exampleDir := filepath.Join(root, "examples", "admin-console")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ingressFile := filepath.Join(exampleDir, "ingress.yaml")
	if err := os.WriteFile(ingressFile, []byte(`version: v1
logging:
  enable: true
  transports:
    - type: file
      path: ./access.log
      levels:
        error: ./error.log
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exampleDir, "access.log"), []byte("sample line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{IngressConfigPath: ingressFile}
	if err := ResolvePaths(&cfg, ingressFile); err != nil {
		t.Fatal(err)
	}
	wantAccess := filepath.Join(exampleDir, "access.log")
	wantError := filepath.Join(exampleDir, "error.log")
	if cfg.AccessLogPath != wantAccess {
		t.Fatalf("access_log_path: got %q want %q", cfg.AccessLogPath, wantAccess)
	}
	if cfg.ErrorLogPath != wantError {
		t.Fatalf("error_log_path: got %q want %q", cfg.ErrorLogPath, wantError)
	}
}
