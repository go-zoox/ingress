package rbac

import (
	"path/filepath"
	"testing"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

func TestListNavigation(t *testing.T) {
	dir := t.TempDir()
	dsn := filepath.Join(dir, "rbac-menu.db")
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

	all, err := svc.ListNavigation("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Groups) != 6 {
		t.Fatalf("expected 6 nav groups without filter, got %d", len(all.Groups))
	}

	adminNav, err := svc.ListNavigation(defaultAdminUsername)
	if err != nil {
		t.Fatal(err)
	}
	if len(adminNav.Groups) != 6 {
		t.Fatalf("expected admin to see all groups, got %d", len(adminNav.Groups))
	}

	perms, err := svc.ListPermissions()
	if err != nil {
		t.Fatal(err)
	}
	var overviewMenuID uint
	for _, p := range perms {
		if p.Code == MenuPermissionCode("overview") {
			overviewMenuID = p.ID
			break
		}
	}
	if overviewMenuID == 0 {
		t.Fatal("menu:overview permission missing")
	}

	role, err := svc.CreateRole(RoleInput{
		Code:          "menu-only",
		Name:          "仅总览菜单",
		PermissionIDs: []uint{overviewMenuID},
	})
	if err != nil {
		t.Fatal(err)
	}
	user, err := svc.CreateUser(UserInput{
		Username:    "menu-user",
		DisplayName: "菜单用户",
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
	if len(nav.Groups) != 1 || nav.Groups[0].Label != "监控" || len(nav.Groups[0].Items) != 1 {
		t.Fatalf("unexpected filtered nav: %+v", nav.Groups)
	}
	if nav.Groups[0].Items[0].To != "/" {
		t.Fatalf("expected overview link, got %+v", nav.Groups[0].Items[0])
	}
}

func TestMenuPermissionsSynced(t *testing.T) {
	menuCount := len(BuiltinMenus())
	permCount := len(MenuPermissions())
	if menuCount != permCount {
		t.Fatalf("menu items and menu permissions mismatch: %d vs %d", menuCount, permCount)
	}
}
