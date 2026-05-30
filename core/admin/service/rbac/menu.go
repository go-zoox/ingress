package rbac

import (
	"errors"

	"github.com/go-zoox/ingress/core/admin/model"
	"gorm.io/gorm"
)

type menuItemDef struct {
	Key      string
	To       string
	Label    string
	Icon     string
	Group    string
	End      bool
	BadgeKey string
}

// MenuPermissionCode returns the RBAC code gating sidebar visibility.
func MenuPermissionCode(key string) string {
	return "menu:" + key
}

// BuiltinMenus returns the Admin Console sidebar catalog.
func BuiltinMenus() []menuItemDef {
	return []menuItemDef{
		{Key: "overview", To: "/", Label: "总览", Icon: "layout-dashboard", Group: "监控", End: true},
		{Key: "events", To: "/events", Label: "事件", Icon: "activity", Group: "监控", BadgeKey: "events"},
		{Key: "investigate", To: "/investigate", Label: "调查", Icon: "search", Group: "监控"},
		{Key: "logs", To: "/logs", Label: "日志", Icon: "scroll-text", Group: "监控"},

		{Key: "routes", To: "/routes", Label: "路由", Icon: "arrow-left-right", Group: "流量"},
		{Key: "services", To: "/services", Label: "服务", Icon: "server", Group: "流量"},
		{Key: "cache", To: "/cache", Label: "缓存", Icon: "hard-drive", Group: "流量"},

		{Key: "waf", To: "/waf", Label: "WAF", Icon: "shield", Group: "安全"},
		{Key: "tls", To: "/tls", Label: "TLS", Icon: "lock", Group: "安全", BadgeKey: "tls"},
		{Key: "healths", To: "/healths", Label: "健康检查", Icon: "heart-pulse", Group: "安全", BadgeKey: "healths"},

		{Key: "maintenance", To: "/maintenance", Label: "维护模式", Icon: "construction", Group: "维护"},
		{Key: "jobs", To: "/jobs", Label: "定时任务", Icon: "clock", Group: "维护"},
		{Key: "terminal", To: "/terminal", Label: "Web 终端", Icon: "terminal", Group: "维护"},

		{Key: "rbac-users", To: "/rbac/users", Label: "用户管理", Icon: "users", Group: "权限"},
		{Key: "rbac-roles", To: "/rbac/roles", Label: "角色管理", Icon: "user-cog", Group: "权限"},
		{Key: "rbac-permissions", To: "/rbac/permissions", Label: "权限管理", Icon: "key-round", Group: "权限"},

		{Key: "config", To: "/config", Label: "配置", Icon: "file-code-2", Group: "系统"},
		{Key: "settings", To: "/settings", Label: "设置", Icon: "settings", Group: "系统"},
	}
}

// MenuPermissions returns builtin menu visibility grants derived from BuiltinMenus.
func MenuPermissions() []permissionDef {
	items := BuiltinMenus()
	out := make([]permissionDef, 0, len(items))
	for _, item := range items {
		out = append(out, permissionDef{
			Code:        MenuPermissionCode(item.Key),
			Name:        "菜单：" + item.Label,
			Group:       "菜单",
			Description: "侧栏显示「" + item.Group + " / " + item.Label + "」",
		})
	}
	return out
}

// AllBuiltinPermissions merges action and menu builtin permissions.
func AllBuiltinPermissions() []permissionDef {
	out := make([]permissionDef, 0, len(BuiltinPermissions())+len(MenuPermissions()))
	out = append(out, BuiltinPermissions()...)
	out = append(out, MenuPermissions()...)
	return out
}

var menuGroupOrder = []string{"监控", "流量", "安全", "维护", "权限", "系统"}

// NavItemRow is a sidebar link returned to the Admin UI.
type NavItemRow struct {
	To         string `json:"to"`
	Label      string `json:"label"`
	Icon       string `json:"icon"`
	End        bool   `json:"end,omitempty"`
	BadgeKey   string `json:"badge_key,omitempty"`
	Permission string `json:"permission"`
}

// NavGroupRow groups sidebar links under a section label.
type NavGroupRow struct {
	Label string       `json:"label"`
	Items []NavItemRow `json:"items"`
}

// NavigationResult is the filtered sidebar tree for a user.
type NavigationResult struct {
	Username string        `json:"username,omitempty"`
	Groups   []NavGroupRow `json:"groups"`
}

// ListNavigation returns sidebar groups filtered by the user's menu:* permissions.
// Action grants (e.g. routes:read) alone do not show a menu item.
// When username is empty (auth disabled), all menu items are returned.
func (s *Service) ListNavigation(username string) (NavigationResult, error) {
	username = normalizeUsername(username)
	codes, err := s.PermissionCodesForUser(username)
	if err != nil {
		return NavigationResult{}, err
	}
	filter := username != ""

	groupMap := make(map[string][]NavItemRow)
	for _, def := range BuiltinMenus() {
		perm := MenuPermissionCode(def.Key)
		if filter {
			if _, ok := codes[perm]; !ok {
				continue
			}
		}
		groupMap[def.Group] = append(groupMap[def.Group], NavItemRow{
			To:         def.To,
			Label:      def.Label,
			Icon:       def.Icon,
			End:        def.End,
			BadgeKey:   def.BadgeKey,
			Permission: perm,
		})
	}

	groups := make([]NavGroupRow, 0, len(menuGroupOrder))
	for _, label := range menuGroupOrder {
		items := groupMap[label]
		if len(items) == 0 {
			continue
		}
		groups = append(groups, NavGroupRow{Label: label, Items: items})
	}
	return NavigationResult{Username: username, Groups: groups}, nil
}

// PermissionCodesForUser resolves effective permission codes from enabled roles.
func (s *Service) PermissionCodesForUser(username string) (map[string]struct{}, error) {
	username = normalizeUsername(username)
	if username == "" {
		return nil, nil
	}
	var user model.RBACUser
	err := db().Preload("Roles.Permissions").Where("username = ? AND enabled = ?", username, true).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	codes := make(map[string]struct{})
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			codes[perm.Code] = struct{}{}
		}
	}
	return codes, nil
}
