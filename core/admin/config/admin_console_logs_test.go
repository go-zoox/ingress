package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/service"
)

func TestAdminConsoleDemoLogPaths(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for root != "" && !strings.HasSuffix(root, "ingress") {
		root = filepath.Dir(root)
	}
	ingressFile := filepath.Join(root, "examples", "admin-console", "ingress.yaml")
	if _, err := os.Stat(ingressFile); err != nil {
		t.Skip("examples/admin-console not found")
	}

	cfg := &admincfg.Config{IngressConfigPath: ingressFile}
	if err := admincfg.ResolvePaths(cfg, ingressFile); err != nil {
		t.Fatal(err)
	}
	wantAccess := filepath.Join(filepath.Dir(ingressFile), "access.log")
	if cfg.AccessLogPath != wantAccess {
		t.Fatalf("access_log_path: got %q want %q", cfg.AccessLogPath, wantAccess)
	}

	m := service.NewMetrics(service.NewLogs(cfg), nil)
	out := m.Overview("15m")
	t.Logf("access=%s source=%s total=%d", cfg.AccessLogPath, out.Source, out.Total)
	if out.Source == "access_log_empty" {
		t.Fatalf("expected parsed access log, got source=%q total=%d", out.Source, out.Total)
	}
	if out.Total == 0 {
		t.Fatalf("expected non-zero total from sample access.log, got %d source=%q stale=%v", out.Total, out.Source, out.WindowStale)
	}
}