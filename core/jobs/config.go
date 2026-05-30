package jobs

import "strings"

// Config is the top-level jobs section in ingress.yaml.
type Config struct {
	Builtins map[string]BuiltinOverride `config:"builtins" yaml:"builtins,omitempty"`
	Items    []Item                     `config:"items" yaml:"items,omitempty"`
}

// BuiltinOverride adjusts a built-in job (schedule, enabled, params).
type BuiltinOverride struct {
	Enabled  *bool     `config:"enabled" yaml:"enabled,omitempty"`
	Schedule string    `config:"schedule" yaml:"schedule,omitempty"`
	Params   JobParams `config:"params" yaml:"params,omitempty"`
}

// Item is a user-defined scheduled job.
type Item struct {
	ID         string    `config:"id" yaml:"id"`
	Name       string    `config:"name" yaml:"name"`
	Kind       string    `config:"kind" yaml:"kind"` // http_call | script (command is legacy alias)
	Schedule   string    `config:"schedule" yaml:"schedule"`
	Enabled    bool      `config:"enabled" yaml:"enabled"`
	TimeoutSec int64     `config:"timeout_sec" yaml:"timeout_sec,omitempty"`
	OnFailure  string    `config:"on_failure" yaml:"on_failure,omitempty"` // log | retry | disable
	Params     JobParams `config:"params" yaml:"params"`
}

// JobParams holds type-specific fields for custom and built-in jobs.
type JobParams struct {
	// http_call
	Method        string            `config:"method" yaml:"method,omitempty"`
	URL           string            `config:"url" yaml:"url,omitempty"`
	Headers       map[string]string `config:"headers" yaml:"headers,omitempty"`
	Body          string            `config:"body" yaml:"body,omitempty"`
	ExpectStatus  []int64           `config:"expect_status" yaml:"expect_status,omitempty"`
	InsecureTLS   bool              `config:"insecure_tls" yaml:"insecure_tls,omitempty"`
	// script
	Engine  string            `config:"engine" yaml:"engine,omitempty"` // shell | javascript | go
	Script  string            `config:"script" yaml:"script,omitempty"`
	Shell   string            `config:"shell" yaml:"shell,omitempty"` // only when engine=shell (interpreter: sh, bash, …)
	Workdir string            `config:"workdir" yaml:"workdir,omitempty"`
	Env     map[string]string `config:"env" yaml:"env,omitempty"`
	// legacy script/command fields (read-only compat; cleared on save via PrepareScriptParams)
	Command string   `config:"command" yaml:"command,omitempty"`
	Args    []string `config:"args" yaml:"args,omitempty"`
	// built-in: purge_waf_events / purge_audit_logs
	RetainDays int `config:"retain_days" yaml:"retain_days,omitempty"`
}

// AdminJobs configures platform job execution policy under admin.jobs.
type AdminJobs struct {
	// AllowCommand: omitted or true enables command jobs; false disables.
	AllowCommand          *bool    `config:"allow_command" yaml:"allow_command,omitempty"`
	CommandAllowlist      []string `config:"command_allowlist" yaml:"command_allowlist,omitempty"`
	CommandWorkdir        string   `config:"command_workdir" yaml:"command_workdir,omitempty"`
	CommandMaxOutputBytes int64    `config:"command_max_output_bytes,default=65536" yaml:"command_max_output_bytes,omitempty"`
}

// CommandExecutionEnabled reports whether command jobs may be created and run.
func (p AdminJobs) CommandExecutionEnabled() bool {
	if p.AllowCommand == nil {
		return true
	}
	return *p.AllowCommand
}

// CommandRestricted reports whether only command_allowlist entries may run.
func (p AdminJobs) CommandRestricted() bool {
	return len(p.CommandAllowlist) > 0
}

const (
	KindHTTPCall = "http_call"
	KindScript   = "script"
	KindCommand  = "command" // legacy alias for KindScript

	OnFailureLog     = "log"
	OnFailureRetry   = "retry"
	OnFailureDisable = "disable"
)

// NormalizeJobKind maps legacy kind values to their canonical form.
func NormalizeJobKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case KindCommand, KindScript:
		return KindScript
	default:
		return strings.TrimSpace(kind)
	}
}

// IsScriptKind reports whether kind runs a host script job.
func IsScriptKind(kind string) bool {
	return NormalizeJobKind(kind) == KindScript
}
