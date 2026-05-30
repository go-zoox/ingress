package scriptexec

import (
	"context"
	"fmt"

	ingjobs "github.com/go-zoox/ingress/core/jobs"
)

// Options configures embedded script execution.
type Options struct {
	MaxOutputBytes int64
	Workdir        string
	Env            map[string]string
}

// Run executes a script job and returns captured log output.
func Run(ctx context.Context, engine, script string, opts Options) (string, error) {
	switch ingjobs.NormalizeScriptEngine(engine) {
	case ingjobs.ScriptEngineJavaScript:
		return runJavaScript(ctx, script, opts)
	case ingjobs.ScriptEngineGo:
		return runGo(ctx, script, opts)
	default:
		return "", fmt.Errorf("unsupported script engine %q", engine)
	}
}
