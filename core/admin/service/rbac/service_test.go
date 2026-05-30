package rbac

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

func TestRBACSeedAndCRUD(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "rbac.db")
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
	if len(perms) < 40 {
		t.Fatalf("expected builtin permissions including menu grants, got %d", len(perms))
	}

	roles, err := svc.ListRoles()
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 5 {
		t.Fatalf("expected 5 builtin roles, got %d", len(roles))
	}

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Username != defaultAdminUsername {
		t.Fatalf("unexpected default user: %+v", users)
	}

	customPerm, err := svc.CreatePermission(PermissionInput{
		Code: "custom:action",
		Name: "自定义动作",
		Group: "自定义",
	})
	if err != nil {
		t.Fatal(err)
	}

	role, err := svc.CreateRole(RoleInput{
		Code:          "ops",
		Name:          "运维",
		Description:   "自定义运维角色",
		PermissionIDs: []uint{customPerm.ID, perms[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(role.PermissionIDs) != 2 {
		t.Fatalf("expected 2 permissions on role, got %+v", role.PermissionIDs)
	}

	user, err := svc.CreateUser(UserInput{
		Username:    "ops-user",
		DisplayName: "运维同学",
		Password:    "secret1",
		Enabled:     true,
		RoleIDs:     []uint{role.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.RoleIDs) != 1 {
		t.Fatalf("expected role assignment, got %+v", user.RoleIDs)
	}

	if err := svc.UpdateUserPassword(user.ID, PasswordInput{Password: "secret2"}); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteUser(user.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRole(role.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeletePermission(customPerm.ID); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureBasicAuthUserIsSuperAdmin(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "rbac-basic-admin.db")
	if err := gormx.LoadDB("sqlite", dsn); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}

	svc := New()
	opts := SeedOptions{BasicUsername: "root", BasicPassword: "s3cret"}
	if err := svc.Seed(opts); err != nil {
		t.Fatal(err)
	}

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Username != "root" {
		t.Fatalf("unexpected users after seed: %+v", users)
	}
	roles, err := svc.ListRoles()
	if err != nil {
		t.Fatal(err)
	}
	var adminRoleID uint
	for _, role := range roles {
		if role.Code == adminRoleCode {
			adminRoleID = role.ID
			break
		}
	}
	if adminRoleID == 0 {
		t.Fatal("admin role missing")
	}
	if len(users[0].RoleIDs) != 1 || users[0].RoleIDs[0] != adminRoleID {
		t.Fatalf("expected root user to have admin role, got role_ids=%v", users[0].RoleIDs)
	}

	viewerRole := roles
	var viewerID uint
	for _, role := range viewerRole {
		if role.Code == viewerRoleCode {
			viewerID = role.ID
			break
		}
	}
	if viewerID == 0 {
		t.Fatal("viewer role missing")
	}
	if _, err := svc.UpdateUser(users[0].ID, UserInput{
		Username:    "root",
		DisplayName: users[0].DisplayName,
		Enabled:     true,
		RoleIDs:     []uint{viewerID},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.CreateUser(UserInput{
		Username:    "ops-user",
		DisplayName: "运维",
		Password:    "secret1",
		Enabled:     true,
		RoleIDs:     []uint{viewerID},
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Seed(opts); err != nil {
		t.Fatal(err)
	}

	users, err = svc.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	var root UserRow
	for _, user := range users {
		if user.Username == "root" {
			root = user
			break
		}
	}
	if root.Username == "" {
		t.Fatal("root user missing after re-seed")
	}
	if len(root.RoleIDs) != 2 {
		t.Fatalf("expected root to keep viewer and regain admin role, got role_ids=%v", root.RoleIDs)
	}
	hasAdmin := false
	for _, id := range root.RoleIDs {
		if id == adminRoleID {
			hasAdmin = true
			break
		}
	}
	if !hasAdmin {
		t.Fatalf("expected admin role on root user, got role_ids=%v", root.RoleIDs)
	}
}

func TestRBACProtectBuiltin(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "rbac-protect.db")
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
	if _, err := svc.UpdatePermission(perms[0].ID, PermissionInput{Name: "x"}); err == nil {
		t.Fatal("expected builtin permission edit to fail")
	}
	if err := svc.DeletePermission(perms[0].ID); err == nil {
		t.Fatal("expected builtin permission delete to fail")
	}

	roles, err := svc.ListRoles()
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRole(roles[0].ID); err == nil {
		t.Fatal("expected builtin role delete to fail")
	}

	users, err := svc.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(users[0].ID); err == nil {
		t.Fatal("expected builtin user delete to fail")
	}
}
