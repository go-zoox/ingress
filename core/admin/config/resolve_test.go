package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths_relativeToIngressConfigDir(t *testing.T) {
	root := t.TempDir()
	exampleDir := filepath.Join(root, "examples", "basic")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ingressFile := filepath.Join(exampleDir, "ingress.yaml")
	if err := os.WriteFile(ingressFile, []byte("version: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		IngressConfigPath: ingressFile,
		PidFile:           "/tmp/gozoox.ingress.pid",
		Database: Database{
			DSN: "file:./admin.db?cache=shared",
		},
	}
	if err := ResolvePaths(&cfg, ingressFile); err != nil {
		t.Fatal(err)
	}
	wantConfig, err := absIngressConfigFile(ingressFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IngressConfigPath != wantConfig {
		t.Fatalf("config_path: got %q want %q", cfg.IngressConfigPath, wantConfig)
	}
	if cfg.PidFile != "/tmp/gozoox.ingress.pid" {
		t.Fatalf("pid_file: %q", cfg.PidFile)
	}
	wantDB := filepath.Join(exampleDir, "admin.db")
	if cfg.Database.DSN != "file:"+wantDB+"?cache=shared" {
		t.Fatalf("dsn: got %q want file:%s?cache=shared", cfg.Database.DSN, wantDB)
	}
}

func TestResolvePaths_cliRelativePathNotDoubled(t *testing.T) {
	root := t.TempDir()
	exampleDir := filepath.Join(root, "examples", "admin-console")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ingressFile := filepath.Join(exampleDir, "ingress.yaml")
	if err := os.WriteFile(ingressFile, []byte("version: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	rel := filepath.Join("examples", "admin-console", "ingress.yaml")
	cfg := Config{
		IngressConfigPath: rel,
	}
	if err := ResolvePaths(&cfg, rel); err != nil {
		t.Fatal(err)
	}
	if cfg.IngressConfigPath != ingressFile {
		t.Fatalf("config_path: got %q want %q", cfg.IngressConfigPath, ingressFile)
	}
}
