package core

import (
	"path/filepath"
	"testing"

	zcfg "github.com/go-zoox/zoox/config"
)

func TestResolveConfigPaths_loggerFiles(t *testing.T) {
	cfg := &Config{
		Logger: zcfg.Logger{
			Level: "warn",
			Transports: []zcfg.Transport{
				{
					Type: "file",
					Path: "./access.log",
					Levels: map[string]string{
						"error": "./error.log",
					},
				},
			},
		},
	}
	configFile := filepath.Join("examples", "admin-console", "ingress.yaml")
	if err := ResolveConfigPaths(cfg, configFile); err != nil {
		t.Fatal(err)
	}
	wantBase, err := ingressConfigDir(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Logger.Transports[0].Path != filepath.Join(wantBase, "access.log") {
		t.Fatalf("access log path: got %q want %q", cfg.Logger.Transports[0].Path, filepath.Join(wantBase, "access.log"))
	}
	if cfg.Logger.Transports[0].Levels["error"] != filepath.Join(wantBase, "error.log") {
		t.Fatalf("error log path: got %q want %q", cfg.Logger.Transports[0].Levels["error"], filepath.Join(wantBase, "error.log"))
	}
}

func TestResolveConfigPaths_keepsAbsolutePaths(t *testing.T) {
	cfg := &Config{
		Logger: zcfg.Logger{
			Transports: []zcfg.Transport{
				{Type: "file", Path: "/var/log/ingress/access.log"},
			},
		},
	}
	if err := ResolveConfigPaths(cfg, "examples/admin-console/ingress.yaml"); err != nil {
		t.Fatal(err)
	}
	if cfg.Logger.Transports[0].Path != "/var/log/ingress/access.log" {
		t.Fatalf("got %q", cfg.Logger.Transports[0].Path)
	}
}
