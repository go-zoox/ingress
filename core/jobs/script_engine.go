package jobs

import "strings"

const (
	ScriptEngineShell      = "shell"
	ScriptEngineJavaScript = "javascript"
	ScriptEngineGo         = "go"
	DefaultScriptEngine    = ScriptEngineShell
)

// NormalizeScriptEngine returns the canonical script engine name.
func NormalizeScriptEngine(engine string) string {
	switch strings.TrimSpace(strings.ToLower(engine)) {
	case "", ScriptEngineShell:
		return ScriptEngineShell
	case ScriptEngineJavaScript, "js":
		return ScriptEngineJavaScript
	case ScriptEngineGo, "golang":
		return ScriptEngineGo
	default:
		return strings.TrimSpace(engine)
	}
}

// IsEmbeddedScriptEngine reports whether the engine runs in-process (javascript/go).
func IsEmbeddedScriptEngine(engine string) bool {
	switch NormalizeScriptEngine(engine) {
	case ScriptEngineJavaScript, ScriptEngineGo:
		return true
	default:
		return false
	}
}
