package jobs

import "testing"

func TestAdminJobs_CommandExecutionDefault(t *testing.T) {
	var p AdminJobs
	if !p.CommandExecutionEnabled() {
		t.Fatal("expected command execution enabled by default")
	}
	if p.CommandRestricted() {
		t.Fatal("expected unrestricted by default")
	}
}

func TestAdminJobs_CommandDisabledExplicit(t *testing.T) {
	disabled := false
	p := AdminJobs{AllowCommand: &disabled}
	if p.CommandExecutionEnabled() {
		t.Fatal("expected disabled")
	}
}

func TestAdminJobs_CommandRestricted(t *testing.T) {
	p := AdminJobs{CommandAllowlist: []string{"/bin/echo"}}
	if !p.CommandRestricted() {
		t.Fatal("expected restricted")
	}
}

func TestNormalizeJobKind(t *testing.T) {
	tests := map[string]string{
		"script":  KindScript,
		"command": KindScript,
		" http_call ": KindHTTPCall,
		"":          "",
	}
	for in, want := range tests {
		if got := NormalizeJobKind(in); got != want {
			t.Fatalf("NormalizeJobKind(%q) = %q, want %q", in, got, want)
		}
	}
	if !IsScriptKind("command") || !IsScriptKind("script") || IsScriptKind("http_call") {
		t.Fatal("IsScriptKind mismatch")
	}
}
