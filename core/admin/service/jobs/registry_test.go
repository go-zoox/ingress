package jobs

import (
	"testing"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
)

func TestValidateCustomItem_HTTPCall(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "ping",
		Kind:     ingjobs.KindHTTPCall,
		Schedule: "0 * * * *",
		Params:   ingjobs.JobParams{URL: "http://127.0.0.1/health"},
	}
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{}); err != nil {
		t.Fatal(err)
	}
	if item.Params.Method != "GET" {
		t.Fatalf("method = %q", item.Params.Method)
	}
}

func TestValidateCustomItem_CommandRequiresAllow(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "validate",
		Kind:     ingjobs.KindScript,
		Schedule: "@daily",
		Params:   ingjobs.JobParams{Command: "/usr/bin/true"},
	}
	disabled := false
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{AllowCommand: &disabled}); err == nil {
		t.Fatal("expected error when allow_command is false")
	}
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCustomItem_LegacyCommandKind(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "legacy",
		Kind:     ingjobs.KindCommand,
		Schedule: "0 * * * *",
		Params:   ingjobs.JobParams{Command: "/bin/echo"},
	}
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{}); err != nil {
		t.Fatal(err)
	}
	if item.Kind != ingjobs.KindScript {
		t.Fatalf("kind = %q, want script", item.Kind)
	}
}

func TestEffectiveBuiltin(t *testing.T) {
	def, ok := BuiltinByID("purge_waf_events")
	if !ok {
		t.Fatal("missing builtin")
	}
	disabled := false
	enabled, schedule, params := EffectiveBuiltin(def, &ingjobs.BuiltinOverride{
		Enabled:  &disabled,
		Schedule: "0 5 * * *",
		Params:   ingjobs.JobParams{RetainDays: 7},
	})
	if enabled {
		t.Fatal("expected disabled")
	}
	if schedule != "0 5 * * *" {
		t.Fatalf("schedule = %q", schedule)
	}
	if params.RetainDays != 7 {
		t.Fatalf("retain_days = %d", params.RetainDays)
	}
}

func TestBuiltinByID_NormalizesPrefix(t *testing.T) {
	if _, ok := BuiltinByID("builtin.purge_waf_events"); !ok {
		t.Fatal("expected builtin with prefix")
	}
}

func TestCommandAllowed(t *testing.T) {
	list := []string{"/bin/echo", "/usr/bin/true"}
	if !commandAllowed("/bin/echo", list) {
		t.Fatal("expected allowed")
	}
	if commandAllowed("/bin/sh", list) {
		t.Fatal("expected denied")
	}
}

func TestMergeParams(t *testing.T) {
	base := ingjobs.JobParams{RetainDays: 30, Method: "GET"}
	patch := ingjobs.JobParams{RetainDays: 7, URL: "http://x", Headers: map[string]string{"A": "B"}}
	out := mergeParams(base, patch)
	if out.RetainDays != 7 || out.URL != "http://x" || out.Method != "GET" {
		t.Fatalf("out = %+v", out)
	}
}

func TestDefaultOnFailure(t *testing.T) {
	if DefaultOnFailure("") != ingjobs.OnFailureLog {
		t.Fatal("expected log default")
	}
	if DefaultOnFailure("retry") != ingjobs.OnFailureRetry {
		t.Fatal("expected retry")
	}
	if DefaultOnFailure("unknown") != ingjobs.OnFailureLog {
		t.Fatal("expected fallback log")
	}
}

func TestCronKey(t *testing.T) {
	if got := CronKey(SourceBuiltin, "sync_geoip"); got != "job:builtin:sync_geoip" {
		t.Fatalf("key = %q", got)
	}
}

func TestValidateCustomItem_CommandAllowlist(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "sh",
		Kind:     ingjobs.KindScript,
		Schedule: "0 * * * *",
		Params:   ingjobs.JobParams{Command: "/bin/sh"},
	}
	policy := ingjobs.AdminJobs{
		CommandAllowlist: []string{"/bin/echo"},
	}
	if err := ValidateCustomItem(item, policy); err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestValidateCustomItem_ShellNotAllowedOnJavaScript(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "js1",
		Kind:     ingjobs.KindScript,
		Schedule: "0 * * * *",
		Params: ingjobs.JobParams{
			Engine: ingjobs.ScriptEngineJavaScript,
			Script: "console.log(1)",
			Shell:  "bash",
		},
	}
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{}); err == nil {
		t.Fatal("expected error when shell set on javascript engine")
	}
}

func TestValidateCustomItem_GoEmbedded(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "go1",
		Kind:     ingjobs.KindScript,
		Schedule: "0 * * * *",
		Params: ingjobs.JobParams{
			Engine: ingjobs.ScriptEngineGo,
			Script: `import "fmt"
fmt.Println("ok")`,
		},
	}
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{}); err != nil {
		t.Fatal(err)
	}
	if item.Params.Shell != "" {
		t.Fatalf("shell = %q", item.Params.Shell)
	}
}

func TestValidateCustomItem_ShellEchoScript(t *testing.T) {
	item := &ingjobs.Item{
		ID:       "sh-echo",
		Kind:     ingjobs.KindScript,
		Schedule: "0 * * * *",
		Params: ingjobs.JobParams{
			Engine: ingjobs.ScriptEngineShell,
			Shell:  "sh",
			Script: `echo jobs-demo`,
		},
	}
	policy := ingjobs.AdminJobs{CommandAllowlist: []string{"/bin/sh"}}
	if err := ValidateCustomItem(item, policy); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCustomItem_UnsupportedKind(t *testing.T) {
	item := &ingjobs.Item{ID: "x", Kind: "unknown", Schedule: "* * * * *"}
	if err := ValidateCustomItem(item, ingjobs.AdminJobs{}); err == nil {
		t.Fatal("expected kind error")
	}
}
