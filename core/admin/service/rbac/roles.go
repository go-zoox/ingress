package rbac

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/admin/model"
)

const (
	adminRoleCode     = "admin"
	viewerRoleCode    = "viewer"
	operatorRoleCode  = "operator"
	developerRoleCode = "developer"
	securityRoleCode  = "security"
)

type builtinRoleDef struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

func menuPerm(key string) string {
	return MenuPermissionCode(key)
}

func viewerPermissionCodes() []string {
	return []string{
		"overview:read",
		"events:read",
		"investigate:read",
		"logs:read",
		"routes:read",
		"services:read",
		"cache:read",
		"waf:read",
		"tls:read",
		"health:read",
		menuPerm("overview"),
		menuPerm("events"),
		menuPerm("investigate"),
		menuPerm("logs"),
		menuPerm("routes"),
		menuPerm("services"),
		menuPerm("cache"),
		menuPerm("waf"),
		menuPerm("tls"),
		menuPerm("healths"),
	}
}

func operatorPermissionCodes() []string {
	out := append([]string(nil), viewerPermissionCodes()...)
	out = append(out,
		"maintenance:read",
		"maintenance:write",
		"jobs:read",
		"jobs:write",
		"terminal:use",
		menuPerm("maintenance"),
		menuPerm("jobs"),
		menuPerm("terminal"),
	)
	return out
}

// builtinRoleDefs returns seeded roles with explicit action + menu permissions.
// Sidebar visibility always requires the matching menu:* grant.
func builtinRoleDefs() []builtinRoleDef {
	return []builtinRoleDef{
		{
			Code:        adminRoleCode,
			Name:        "管理员",
			Description: "拥有全部 Admin Console 功能与菜单",
			Permissions: nil, // resolved as all permissions
		},
		{
			Code:        viewerRoleCode,
			Name:        "只读观察",
			Description: "只读查看监控、流量与安全相关页面；不含运维、配置与权限管理菜单",
			Permissions: viewerPermissionCodes(),
		},
		{
			Code:        operatorRoleCode,
			Name:        "运维工程师",
			Description: "只读观察 + 维护、定时任务与 Web 终端",
			Permissions: operatorPermissionCodes(),
		},
		{
			Code:        developerRoleCode,
			Name:        "路由开发",
			Description: "路由、服务、缓存与配置管理；适合平台/网关开发同学",
			Permissions: []string{
				"overview:read",
				"routes:read",
				"routes:write",
				"services:read",
				"services:write",
				"cache:read",
				"cache:write",
				"config:read",
				"config:write",
				"settings:read",
				menuPerm("overview"),
				menuPerm("routes"),
				menuPerm("services"),
				menuPerm("cache"),
				menuPerm("config"),
				menuPerm("settings"),
			},
		},
		{
			Code:        securityRoleCode,
			Name:        "安全工程师",
			Description: "事件、日志、调查与 WAF/TLS/健康检查相关能力",
			Permissions: []string{
				"overview:read",
				"events:read",
				"events:write",
				"investigate:read",
				"logs:read",
				"waf:read",
				"waf:write",
				"tls:read",
				"tls:write",
				"health:read",
				menuPerm("overview"),
				menuPerm("events"),
				menuPerm("investigate"),
				menuPerm("logs"),
				menuPerm("waf"),
				menuPerm("tls"),
				menuPerm("healths"),
			},
		},
	}
}

func resolveRolePermissions(all []model.RBACPermission, codes []string) ([]model.RBACPermission, error) {
	if len(codes) == 0 {
		return append([]model.RBACPermission(nil), all...), nil
	}
	want := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		want[code] = struct{}{}
	}
	out := make([]model.RBACPermission, 0, len(codes))
	for _, perm := range all {
		if _, ok := want[perm.Code]; ok {
			out = append(out, perm)
		}
	}
	if len(out) != len(want) {
		missing := make([]string, 0)
		got := make(map[string]struct{}, len(out))
		for _, perm := range out {
			got[perm.Code] = struct{}{}
		}
		for code := range want {
			if _, ok := got[code]; !ok {
				missing = append(missing, code)
			}
		}
		sortStrings(missing)
		return nil, fmt.Errorf("rbac: missing builtin permissions: %s", strings.Join(missing, ", "))
	}
	return out, nil
}

func sortStrings(items []string) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
