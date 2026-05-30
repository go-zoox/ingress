package rbac

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

func TestBuiltinRolesSeeded(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "rbac-roles.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}

	svc := New()
	if err := svc.Seed(SeedOptions{}); err != nil {
		t.Fatal(err)
	}

	roles, err := svc.ListRoles()
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 5 {
		t.Fatalf("expected 5 builtin roles, got %d", len(roles))
	}

	byCode := map[string]RoleRow{}
	for _, role := range roles {
		byCode[role.Code] = role
	}
	for _, code := range []string{adminRoleCode, viewerRoleCode, operatorRoleCode, developerRoleCode, securityRoleCode} {
		role, ok := byCode[code]
		if !ok {
			t.Fatalf("missing builtin role %q", code)
		}
		if !role.Builtin {
			t.Fatalf("role %q should be builtin", code)
		}
	}

	admin := byCode[adminRoleCode]
	if len(admin.Permissions) < 40 {
		t.Fatalf("admin should have all permissions, got %d", len(admin.Permissions))
	}

	viewer := byCode[viewerRoleCode]
	if !containsPerm(viewer.Permissions, menuPerm("routes")) {
		t.Fatal("viewer should include menu:routes")
	}
	if containsPerm(viewer.Permissions, menuPerm("config")) {
		t.Fatal("viewer should not include menu:config")
	}
	if !containsPerm(viewer.Permissions, "routes:read") {
		t.Fatal("viewer should include routes:read")
	}

	operator := byCode[operatorRoleCode]
	if !containsPerm(operator.Permissions, menuPerm("terminal")) {
		t.Fatal("operator should include menu:terminal")
	}
	if !containsPerm(operator.Permissions, menuPerm("overview")) {
		t.Fatal("operator should inherit viewer menus")
	}
}

func TestActionPermissionWithoutMenuHidden(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "rbac-action-menu.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}

	svc := New()
	if err := svc.Seed(SeedOptions{}); err != nil {
		t.Fatal(err)
	}

	perms, err := svc.ListPermissions()
	if err != nil {
		t.Fatal(err)
	}
	var routesReadID uint
	for _, p := range perms {
		if p.Code == "routes:read" {
			routesReadID = p.ID
			break
		}
	}
	if routesReadID == 0 {
		t.Fatal("routes:read permission missing")
	}

	role, err := svc.CreateRole(RoleInput{
		Code:          "routes-read-only",
		Name:          "仅路由读权限",
		PermissionIDs: []uint{routesReadID},
	})
	if err != nil {
		t.Fatal(err)
	}
	user, err := svc.CreateUser(UserInput{
		Username:    "routes-read-user",
		DisplayName: "路由只读",
		Password:    "secret1",
		Enabled:     true,
		RoleIDs:     []uint{role.ID},
	})
	if err != nil {
		t.Fatal(err)
	}

	nav, err := svc.ListNavigation(user.Username)
	if err != nil {
		t.Fatal(err)
	}
	if len(nav.Groups) != 0 {
		t.Fatalf("expected no menus without menu:* permission, got %+v", nav.Groups)
	}
	_ = role
}

func containsPerm(codes []string, want string) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}
