package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/admin/service/rbac"
	"github.com/go-zoox/zoox"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

func testAdminConfig(t *testing.T) (*admincfg.Config, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &admincfg.Config{
		Enabled:           true,
		Port:              9080,
		IngressConfigPath: filepath.Join(dir, "ingress.yaml"),
		Database: admincfg.Database{
			Driver: "sqlite",
			DSN:    "file:" + filepath.Join(dir, "admin.db") + "?cache=shared&_fk=1",
		},
		Web: admincfg.Web{DevProxy: true},
		Auth: admincfg.Auth{
			Type: "basic",
			Basic: admincfg.AuthBasic{
				Username: "admin",
				Password: "admin",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	return cfg, dir
}

func startTestAdminServer(t *testing.T, cfg *admincfg.Config) (*httptest.Server, *http.Client) {
	t.Helper()
	zooxApp, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(zooxApp)
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return ts, &http.Client{Jar: jar}
}

func postJSON(t *testing.T, client *http.Client, url, body string) (*http.Response, apiEnvelope) {
	t.Helper()
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode envelope: %v body=%s", err, raw)
	}
	return resp, env
}

func TestApplyServeDefaultsSetsSecretKey(t *testing.T) {
	app := zoox.New()
	if app.Config.SecretKey != "" {
		t.Fatalf("expected empty secret before defaults, got %q", app.Config.SecretKey)
	}
	applyServeDefaults(app)
	if app.Config.SecretKey == "" {
		t.Fatal("expected SecretKey after applyServeDefaults")
	}
	if app.Config.Session.MaxAge == 0 {
		t.Fatal("expected session max age after applyServeDefaults")
	}
}

func TestAdminSessionPersistsAfterLogin(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)

	loginResp, env := postJSON(t, client, ts.URL+"/api/v1/auth/login", `{"username":"admin","password":"admin"}`)
	if loginResp.StatusCode != http.StatusOK || env.Code != 200 {
		t.Fatalf("login failed status=%d env=%+v", loginResp.StatusCode, env)
	}

	configResp, err := client.Get(ts.URL + "/api/v1/auth/config")
	if err != nil {
		t.Fatal(err)
	}
	defer configResp.Body.Close()
	body, err := io.ReadAll(configResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var configEnv apiEnvelope
	if err := json.Unmarshal(body, &configEnv); err != nil {
		t.Fatal(err)
	}
	if configEnv.Code != 200 {
		t.Fatalf("auth config code=%d message=%q", configEnv.Code, configEnv.Message)
	}
	var result struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(configEnv.Result, &result); err != nil {
		t.Fatal(err)
	}
	if !result.Authenticated {
		t.Fatalf("expected authenticated session, got %s", body)
	}

	menuResp, err := client.Get(ts.URL + "/api/v1/rbac/menus")
	if err != nil {
		t.Fatal(err)
	}
	defer menuResp.Body.Close()
	if menuResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(menuResp.Body)
		t.Fatalf("menus status=%d body=%s", menuResp.StatusCode, body)
	}
}

func TestProtectedAPIRequiresSession(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)

	resp, err := client.Get(ts.URL + "/api/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, status=%d body=%s", resp.StatusCode, body)
	}
}

func TestLoginForbiddenWithoutMenuPermission(t *testing.T) {
	cfg, dir := testAdminConfig(t)
	if err := gormx.LoadDB("sqlite", cfg.Database.DSN); err != nil {
		t.Fatal(err)
	}
	if err := gormx.GetDB().AutoMigrate(model.MigrateModels()...); err != nil {
		t.Fatal(err)
	}
	svc := rbac.New()
	if err := svc.Seed(rbac.SeedOptions{
		BasicUsername: cfg.Auth.Basic.Username,
		BasicPassword: cfg.Auth.Basic.Password,
	}); err != nil {
		t.Fatal(err)
	}
	perms, err := svc.ListPermissions()
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
	if routesReadID == 0 {
		t.Fatal("routes:read permission missing")
	}
	role, err := svc.CreateRole(rbac.RoleInput{
		Code:          "routes-read-no-menu",
		Name:          "路由只读无菜单",
		PermissionIDs: []uint{routesReadID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateUser(rbac.UserInput{
		Username:    "routes-only",
		DisplayName: "无菜单用户",
		Password:    "secret12",
		Enabled:     true,
		RoleIDs:     []uint{role.ID},
	}); err != nil {
		t.Fatal(err)
	}
	_ = dir

	ts, client := startTestAdminServer(t, cfg)
	loginResp, env := postJSON(t, client, ts.URL+"/api/v1/auth/login", `{"username":"routes-only","password":"secret12"}`)
	if loginResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without menu permission, status=%d env=%+v", loginResp.StatusCode, env)
	}
	if env.Message == "" {
		t.Fatalf("expected error message, env=%+v", env)
	}

	configResp, err := client.Get(ts.URL + "/api/v1/auth/config")
	if err != nil {
		t.Fatal(err)
	}
	defer configResp.Body.Close()
	body, _ := io.ReadAll(configResp.Body)
	var configEnv apiEnvelope
	if err := json.Unmarshal(body, &configEnv); err != nil {
		t.Fatal(err)
	}
	var authView struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(configEnv.Result, &authView); err != nil {
		t.Fatal(err)
	}
	if authView.Authenticated {
		t.Fatalf("expected no session after forbidden login, body=%s", body)
	}
}

func TestAdminMenusIncludeOverviewAfterLogin(t *testing.T) {
	cfg, _ := testAdminConfig(t)
	ts, client := startTestAdminServer(t, cfg)

	loginResp, env := postJSON(t, client, ts.URL+"/api/v1/auth/login", `{"username":"admin","password":"admin"}`)
	if loginResp.StatusCode != http.StatusOK || env.Code != 200 {
		t.Fatalf("login failed status=%d env=%+v", loginResp.StatusCode, env)
	}

	menuResp, err := client.Get(ts.URL + "/api/v1/rbac/menus")
	if err != nil {
		t.Fatal(err)
	}
	defer menuResp.Body.Close()
	body, err := io.ReadAll(menuResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var menuEnv apiEnvelope
	if err := json.Unmarshal(body, &menuEnv); err != nil {
		t.Fatal(err)
	}
	var nav struct {
		Groups []struct {
			Items []struct {
				To string `json:"to"`
			} `json:"items"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(menuEnv.Result, &nav); err != nil {
		t.Fatal(err)
	}
	if len(nav.Groups) == 0 {
		t.Fatalf("admin should see menus, got %s", body)
	}
	foundOverview := false
	for _, group := range nav.Groups {
		for _, item := range group.Items {
			if item.To == "/" {
				foundOverview = true
				break
			}
		}
	}
	if !foundOverview {
		t.Fatalf("expected overview menu for admin user, got %s", body)
	}
}
