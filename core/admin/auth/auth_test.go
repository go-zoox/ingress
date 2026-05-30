package auth

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/admin/service/rbac"
)

func TestBasicLogin(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "auth.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}
	if err := rbac.New().Seed(rbac.SeedOptions{}); err != nil {
		t.Fatal(err)
	}

	cfg := &admincfg.Config{
		Auth: admincfg.Auth{Type: "basic"},
	}
	svc := New(cfg, rbac.New())

	if svc.Type() != "basic" {
		t.Fatalf("expected basic auth, got %q", svc.Type())
	}
	if svc.RequiresAuth() != true {
		t.Fatal("basic should require auth")
	}
	if svc.IsPublicPath("GET", "/api/v1/auth/login") != true {
		t.Fatal("login should be public")
	}
	if svc.IsPublicPath("GET", "/api/v1/status") != false {
		t.Fatal("status should be protected")
	}
}

func TestEffectiveAuthType(t *testing.T) {
	if admincfg.EffectiveAuthType("") != "basic" {
		t.Fatal("empty auth type should default to basic")
	}
	if admincfg.EffectiveAuthType("none") != "none" {
		t.Fatal("expected none")
	}
}
