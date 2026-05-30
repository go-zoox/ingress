package jobs

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestJobParams_UnmarshalYAML_LegacyShellBool(t *testing.T) {
	var p JobParams
	if err := yaml.Unmarshal([]byte(`command: echo hi
shell: true`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Shell != DefaultScriptShell {
		t.Fatalf("shell = %q", p.Shell)
	}
}

func TestScriptContent_PrefersScriptField(t *testing.T) {
	got := ScriptContent(JobParams{
		Script:  "echo script",
		Command: "/bin/echo",
		Args:    []string{"legacy"},
	})
	if got != "echo script" {
		t.Fatalf("got %q", got)
	}
}

func TestScriptContent_LegacyCommandArgs(t *testing.T) {
	got := ScriptContent(JobParams{
		Command: "/bin/echo",
		Args:    []string{"hello", "jobs"},
	})
	if got != "/bin/echo hello jobs" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveShellExecutable(t *testing.T) {
	path, err := ResolveShellExecutable("sh")
	if err != nil || path != "/bin/sh" {
		t.Fatalf("sh = %q err=%v", path, err)
	}
	path, err = ResolveShellExecutable("/usr/local/bin/bash")
	if err != nil || path != "/usr/local/bin/bash" {
		t.Fatalf("custom = %q err=%v", path, err)
	}
}

func TestValidateScriptEngineParams_ShellOnlyForShellEngine(t *testing.T) {
	p := &JobParams{
		Engine: ScriptEngineJavaScript,
		Script: "console.log(1)",
		Shell:  "bash",
	}
	if err := ValidateScriptEngineParams(p); err == nil {
		t.Fatal("expected error when shell set on javascript engine")
	}
}

func TestPrepareScriptParams_ClearsShellForEmbeddedEngine(t *testing.T) {
	p := JobParams{
		Engine: ScriptEngineGo,
		Script: `import "fmt"
fmt.Println("ok")`,
		Shell: "bash",
	}
	if err := PrepareScriptParams(&p); err != nil {
		t.Fatal(err)
	}
	if p.Shell != "" {
		t.Fatalf("shell = %q, want empty for go engine", p.Shell)
	}
	if p.Engine != ScriptEngineGo {
		t.Fatalf("engine = %q", p.Engine)
	}
}

func TestValidateScriptEngineParams_UnsupportedEngine(t *testing.T) {
	err := ValidateScriptEngineParams(&JobParams{
		Engine: "python",
		Script: "print(1)",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported script engine") {
		t.Fatalf("err = %v", err)
	}
}

func TestPrepareScriptParams(t *testing.T) {
	p := JobParams{
		Script:  " echo ok \n",
		Shell:   "",
		Command: "legacy",
		Args:    []string{"x"},
	}
	if err := PrepareScriptParams(&p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Script, "echo ok") {
		t.Fatalf("script = %q", p.Script)
	}
	if p.Shell != DefaultScriptShell {
		t.Fatalf("shell = %q", p.Shell)
	}
	if p.Command != "" || len(p.Args) > 0 {
		t.Fatalf("legacy fields not cleared: %+v", p)
	}
}
