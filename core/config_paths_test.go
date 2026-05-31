package core

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
	zcfg "github.com/go-zoox/zoox/config"
)

func TestResolveConfigPaths_loggerFiles(t *testing.T) {
	cfg := &Config{
		Logging: Logging{
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
	if cfg.Logging.Transports[0].Path != filepath.Join(wantBase, "access.log") {
		t.Fatalf("access log path: got %q want %q", cfg.Logging.Transports[0].Path, filepath.Join(wantBase, "access.log"))
	}
	if cfg.Logging.Transports[0].Levels["error"] != filepath.Join(wantBase, "error.log") {
		t.Fatalf("error log path: got %q want %q", cfg.Logging.Transports[0].Levels["error"], filepath.Join(wantBase, "error.log"))
	}
}

func TestResolveConfigPaths_keepsAbsolutePaths(t *testing.T) {
	cfg := &Config{
		Logging: Logging{
			Transports: []zcfg.Transport{
				{Type: "file", Path: "/var/log/ingress/access.log"},
			},
		},
	}
	if err := ResolveConfigPaths(cfg, "examples/admin-console/ingress.yaml"); err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.Transports[0].Path != "/var/log/ingress/access.log" {
		t.Fatalf("got %q", cfg.Logging.Transports[0].Path)
	}
}

func TestResolveConfigPaths_handlerRootDir(t *testing.T) {
	cfg := &Config{
		Rules: []rule.Rule{
			{
				Host: "cdn.example.com",
				Backend: rule.Backend{
					Type: "handler",
					Handler: rule.Handler{
						Type:    "file_server",
						RootDir: "./static",
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
	got := cfg.Rules[0].Backend.Handler.RootDir
	want := filepath.Join(wantBase, "static")
	if got != want {
		t.Fatalf("root_dir: got %q want %q", got, want)
	}
}
