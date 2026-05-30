package rbac

type permissionDef struct {
	Code        string
	Name        string
	Group       string
	Description string
}

// BuiltinPermissions returns the default Admin Console permission catalog.
func BuiltinPermissions() []permissionDef {
	return []permissionDef{
		{Code: "overview:read", Name: "查看总览", Group: "监控", Description: "访问总览面板与指标"},
		{Code: "events:read", Name: "查看事件", Group: "监控", Description: "查看 WAF 与解析异常事件"},
		{Code: "events:write", Name: "处置事件", Group: "监控", Description: "标记或批量更新事件状态"},
		{Code: "investigate:read", Name: "请求调查", Group: "监控", Description: "使用请求调查工具"},
		{Code: "logs:read", Name: "查看日志", Group: "监控", Description: "查看与筛选访问日志"},

		{Code: "routes:read", Name: "查看路由", Group: "流量", Description: "查看路由列表与详情"},
		{Code: "routes:write", Name: "管理路由", Group: "流量", Description: "创建、编辑与发布路由"},
		{Code: "services:read", Name: "查看服务", Group: "流量", Description: "查看 upstream Service 目录"},
		{Code: "services:write", Name: "管理服务", Group: "流量", Description: "编辑 Service 配置"},
		{Code: "cache:read", Name: "查看缓存", Group: "流量", Description: "查看 HTTP 响应缓存策略与统计"},
		{Code: "cache:write", Name: "管理缓存", Group: "流量", Description: "修改缓存相关配置"},

		{Code: "waf:read", Name: "查看 WAF", Group: "安全", Description: "查看 WAF 规则与事件"},
		{Code: "waf:write", Name: "管理 WAF", Group: "安全", Description: "修改 WAF 配置与运行时开关"},
		{Code: "tls:read", Name: "查看 TLS", Group: "安全", Description: "查看证书与 HTTPS 配置"},
		{Code: "tls:write", Name: "管理 TLS", Group: "安全", Description: "修改 TLS 相关配置"},
		{Code: "health:read", Name: "查看健康检查", Group: "安全", Description: "查看后端健康探测结果"},

		{Code: "maintenance:read", Name: "查看维护", Group: "维护", Description: "查看维护模式配置与状态"},
		{Code: "maintenance:write", Name: "管理维护", Group: "维护", Description: "修改维护模式配置"},
		{Code: "jobs:read", Name: "查看定时任务", Group: "维护", Description: "查看调度任务与执行历史"},
		{Code: "jobs:write", Name: "管理定时任务", Group: "维护", Description: "创建、编辑与触发定时任务"},
		{Code: "terminal:use", Name: "使用 Web 终端", Group: "维护", Description: "连接 Admin 主机 Shell"},

		{Code: "rbac:read", Name: "查看权限", Group: "权限", Description: "查看用户、角色与权限"},
		{Code: "rbac:write", Name: "管理权限", Group: "权限", Description: "管理 RBAC 用户、角色与权限"},

		{Code: "config:read", Name: "查看配置", Group: "系统", Description: "查看 ingress.yaml 与版本历史"},
		{Code: "config:write", Name: "管理配置", Group: "系统", Description: "保存、发布与热加载配置"},
		{Code: "settings:read", Name: "查看设置", Group: "系统", Description: "查看 Admin 与集成设置"},
		{Code: "settings:write", Name: "管理设置", Group: "系统", Description: "修改 Admin 与界面偏好"},
	}
}
