package jobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/admin/service"
)

func setupJobRunsDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "jobs.db")
	dsn := "file:" + dbPath + "?cache=shared&_fk=1"
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatalf("load db: %v", err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatalf("migrate admin models: %v", err)
	}
}

func writeTestIngressConfig(t *testing.T, content string) *admincfg.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ingress.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return &admincfg.Config{
		Enabled:           true,
		Port:              9080,
		IngressConfigPath: path,
	}
}

func newTestService(t *testing.T, yamlContent string) *Service {
	t.Helper()
	cfg := writeTestIngressConfig(t, yamlContent)
	ing := service.NewIngress(cfg)
	logs := service.NewLogs(cfg)
	metrics := service.NewMetrics(logs, nil)
	return New(cfg, ing, service.NewAudit(), nil, metrics)
}

const testIngressBase = `port: 8080
admin:
  enabled: true
  port: 9080
  jobs:
    command_allowlist:
      - /bin/sh
      - /bin/echo
rules:
  - host: test.example.com
    backend:
      type: service
      service:
        name: upstream.local
        port: 8080
`
