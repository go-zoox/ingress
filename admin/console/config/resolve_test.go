package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths_relativeToAdminConfigDir(t *testing.T) {
	root := t.TempDir()
	adminDir := filepath.Join(root, "admin")
	exampleDir := filepath.Join(root, "examples", "basic")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ingressFile := filepath.Join(exampleDir, "ingress.yaml")
	if err := os.WriteFile(ingressFile, []byte("version: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	adminFile := filepath.Join(adminDir, "admin.yaml")
	if err := os.MkdirAll(adminDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Ingress: Ingress{
			ConfigPath: "../examples/basic/ingress.yaml",
			PidFile:    "/tmp/gozoox.ingress.pid",
		},
		Database: Database{
			DSN: "file:./admin.db?cache=shared",
		},
	}
	if err := ResolvePaths(&cfg, adminFile); err != nil {
		t.Fatal(err)
	}
	if cfg.Ingress.ConfigPath != ingressFile {
		t.Fatalf("config_path: got %q want %q", cfg.Ingress.ConfigPath, ingressFile)
	}
	if cfg.Ingress.PidFile != "/tmp/gozoox.ingress.pid" {
		t.Fatalf("pid_file: %q", cfg.Ingress.PidFile)
	}
	wantDB := filepath.Join(adminDir, "admin.db")
	if cfg.Database.DSN != "file:"+wantDB+"?cache=shared" {
		t.Fatalf("dsn: got %q want file:%s?cache=shared", cfg.Database.DSN, wantDB)
	}
}
