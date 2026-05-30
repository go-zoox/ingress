package auth

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/admin/service/rbac"
)

func TestEnsureConsoleAccess(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "auth-access.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}
	if err := rbac.New().Seed(rbac.SeedOptions{}); err != nil {
		t.Fatal(err)
	}

	cfg := &admincfg.Config{Auth: admincfg.Auth{Type: "basic"}}
	svc := New(cfg, rbac.New())

	if err := svc.ensureConsoleAccess("admin"); err != nil {
		t.Fatalf("admin should have console access: %v", err)
	}

	perms, err := rbac.New().ListPermissions()
	if err != nil {
		t.Fatal(err)
	}
	var routesReadID uint
	for _, perm := range perms {
		if perm.Code == "routes:read" {
			routesReadID = perm.ID
			break
		}
	}
	role, err := rbac.New().CreateRole(rbac.RoleInput{
		Code:          "no-menu",
		Name:          "无菜单",
		PermissionIDs: []uint{routesReadID},
	})
	if err != nil {
		t.Fatal(err)
	}
	user, err := rbac.New().CreateUser(rbac.UserInput{
		Username:    "no-menu-user",
		DisplayName: "无菜单",
		Password:    "secret12",
		Enabled:     true,
		RoleIDs:     []uint{role.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.ensureConsoleAccess(user.Username); err != ErrNoConsoleAccess {
		t.Fatalf("expected ErrNoConsoleAccess, got %v", err)
	}
}
