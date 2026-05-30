package jobs

import "testing"

func TestNormalizeScriptEngine(t *testing.T) {
	tests := map[string]string{
		"":             ScriptEngineShell,
		"shell":        ScriptEngineShell,
		"javascript":   ScriptEngineJavaScript,
		"js":           ScriptEngineJavaScript,
		"go":           ScriptEngineGo,
		"golang":       ScriptEngineGo,
	}
	for in, want := range tests {
		if got := NormalizeScriptEngine(in); got != want {
			t.Fatalf("NormalizeScriptEngine(%q) = %q, want %q", in, got, want)
		}
	}
	if !IsEmbeddedScriptEngine("javascript") || !IsEmbeddedScriptEngine("go") || IsEmbeddedScriptEngine("shell") {
		t.Fatal("IsEmbeddedScriptEngine mismatch")
	}
}

func TestPrepareScriptParams_EmbeddedEngineClearsShell(t *testing.T) {
	p := JobParams{
		Engine: ScriptEngineJavaScript,
		Script: "console.log(1)",
		Shell:  "bash",
	}
	if err := PrepareScriptParams(&p); err != nil {
		t.Fatal(err)
	}
	if p.Shell != "" {
		t.Fatalf("shell = %q", p.Shell)
	}
	if p.Engine != ScriptEngineJavaScript {
		t.Fatalf("engine = %q", p.Engine)
	}
}
