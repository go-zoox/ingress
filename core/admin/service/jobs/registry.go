package jobs

import (
	"fmt"
	"strings"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
)

const (
	SourceBuiltin = "builtin"
	SourceConfig  = "config"
)

// BuiltinDef describes a built-in ops job registered in code.
type BuiltinDef struct {
	ID              string
	Name            string
	Description     string
	Kind            string
	DefaultSchedule string
	DefaultEnabled  bool
	DefaultParams   ingjobs.JobParams
}

var builtinRegistry = []BuiltinDef{
	{
		ID:              "purge_waf_events",
		Name:            "清理 WAF 事件",
		Description:     "按保留天数删除过期 WAF 事件记录",
		Kind:            "purge_waf_events",
		DefaultSchedule: "0 3 * * *",
		DefaultEnabled:  true,
		DefaultParams:   ingjobs.JobParams{RetainDays: 30},
	},
	{
		ID:              "purge_audit_logs",
		Name:            "清理审计日志",
		Description:     "按保留天数删除过期 Admin 审计日志",
		Kind:            "purge_audit_logs",
		DefaultSchedule: "0 4 * * 0",
		DefaultEnabled:  true,
		DefaultParams:   ingjobs.JobParams{RetainDays: 90},
	},
	{
		ID:              "check_tls_expiry",
		Name:            "TLS 证书检查",
		Description:     "扫描 HTTPS 证书有效期并写入审计日志",
		Kind:            "check_tls_expiry",
		DefaultSchedule: "0 */6 * * *",
		DefaultEnabled:  true,
	},
	{
		ID:              "sync_geoip",
		Name:            "同步 GeoIP",
		Description:     "从 ingress.yaml 重载 MaxMind GeoIP 数据库配置",
		Kind:            "sync_geoip",
		DefaultSchedule: "0 2 * * *",
		DefaultEnabled:  true,
	},
}

func BuiltinByID(id string) (BuiltinDef, bool) {
	id = normalizeBuiltinID(id)
	for _, def := range builtinRegistry {
		if def.ID == id {
			return def, true
		}
	}
	return BuiltinDef{}, false
}

func AllBuiltins() []BuiltinDef {
	out := make([]BuiltinDef, len(builtinRegistry))
	copy(out, builtinRegistry)
	return out
}

func normalizeBuiltinID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "builtin.")
	return id
}

func CronKey(source, id string) string {
	return "job:" + source + ":" + id
}

func ValidateCustomItem(item *ingjobs.Item, policy ingjobs.AdminJobs) error {
	if item == nil {
		return fmt.Errorf("job item is nil")
	}
	item.ID = strings.TrimSpace(item.ID)
	item.Name = strings.TrimSpace(item.Name)
	item.Kind = ingjobs.NormalizeJobKind(item.Kind)
	item.Schedule = strings.TrimSpace(item.Schedule)
	item.OnFailure = strings.TrimSpace(item.OnFailure)
	if item.ID == "" {
		return fmt.Errorf("job id is required")
	}
	if item.Name == "" {
		item.Name = item.ID
	}
	if item.Schedule == "" {
		return fmt.Errorf("job %q: schedule is required", item.ID)
	}
	switch item.Kind {
	case ingjobs.KindHTTPCall:
		return validateHTTPParams(&item.Params)
	case ingjobs.KindScript:
		if !policy.CommandExecutionEnabled() {
			return fmt.Errorf("job %q: script jobs are disabled (admin.jobs.allow_command: false)", item.ID)
		}
		return validateScriptParams(&item.Params, policy)
	default:
		return fmt.Errorf("job %q: unsupported kind %q (use http_call or script)", item.ID, item.Kind)
	}
}

func validateHTTPParams(p *ingjobs.JobParams) error {
	if p == nil || strings.TrimSpace(p.URL) == "" {
		return fmt.Errorf("http_call requires params.url")
	}
	method := strings.ToUpper(strings.TrimSpace(p.Method))
	if method == "" {
		p.Method = "GET"
	} else {
		p.Method = method
	}
	return nil
}

func validateScriptParams(p *ingjobs.JobParams, policy ingjobs.AdminJobs) error {
	if p == nil {
		return fmt.Errorf("script params are nil")
	}
	if strings.TrimSpace(ingjobs.ScriptContent(*p)) == "" {
		return fmt.Errorf("script requires params.script")
	}
	if err := ingjobs.ValidateScriptEngineParams(p); err != nil {
		return err
	}
	engine := ingjobs.NormalizeScriptEngine(p.Engine)
	if engine == ingjobs.ScriptEngineShell {
		shellPath, err := ingjobs.ResolveShellExecutable(ingjobs.ScriptShellName(*p))
		if err != nil {
			return err
		}
		if len(policy.CommandAllowlist) > 0 && !commandAllowed(shellPath, policy.CommandAllowlist) {
			return fmt.Errorf("shell %q is not in admin.jobs.command_allowlist", shellPath)
		}
	}
	return ingjobs.PrepareScriptParams(p)
}

func commandAllowed(command string, allowlist []string) bool {
	command = strings.TrimSpace(command)
	for _, entry := range allowlist {
		if strings.TrimSpace(entry) == command {
			return true
		}
	}
	return false
}

func EffectiveBuiltin(def BuiltinDef, override *ingjobs.BuiltinOverride) (enabled bool, schedule string, params ingjobs.JobParams) {
	enabled = def.DefaultEnabled
	schedule = def.DefaultSchedule
	params = def.DefaultParams
	if override == nil {
		return enabled, schedule, params
	}
	if override.Enabled != nil {
		enabled = *override.Enabled
	}
	if strings.TrimSpace(override.Schedule) != "" {
		schedule = strings.TrimSpace(override.Schedule)
	}
	params = mergeParams(params, override.Params)
	return enabled, schedule, params
}

func mergeParams(base, patch ingjobs.JobParams) ingjobs.JobParams {
	out := base
	if patch.Method != "" {
		out.Method = patch.Method
	}
	if patch.URL != "" {
		out.URL = patch.URL
	}
	if len(patch.Headers) > 0 {
		out.Headers = patch.Headers
	}
	if patch.Body != "" {
		out.Body = patch.Body
	}
	if len(patch.ExpectStatus) > 0 {
		out.ExpectStatus = patch.ExpectStatus
	}
	if patch.InsecureTLS {
		out.InsecureTLS = patch.InsecureTLS
	}
	if patch.Script != "" {
		out.Script = patch.Script
	}
	if patch.Engine != "" {
		out.Engine = patch.Engine
	}
	if patch.Shell != "" {
		out.Shell = patch.Shell
	}
	if patch.Command != "" {
		out.Command = patch.Command
	}
	if len(patch.Args) > 0 {
		out.Args = patch.Args
	}
	if patch.Workdir != "" {
		out.Workdir = patch.Workdir
	}
	if len(patch.Env) > 0 {
		out.Env = patch.Env
	}
	if patch.RetainDays > 0 {
		out.RetainDays = patch.RetainDays
	}
	return out
}

func DefaultTimeoutSec(timeout int64) int64 {
	if timeout <= 0 {
		return 60
	}
	return timeout
}

func DefaultOnFailure(v string) string {
	v = strings.TrimSpace(v)
	switch v {
	case ingjobs.OnFailureRetry, ingjobs.OnFailureDisable:
		return v
	default:
		return ingjobs.OnFailureLog
	}
}
