package scriptexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func runGo(ctx context.Context, script string, opts Options) (string, error) {
	_ = ctx
	maxOut := optsMaxOutput(opts.MaxOutputBytes)

	log, runErr := withCapturedOutput(func() error {
		i := interp.New(interp.Options{
			GoPath: goPath(),
		})
		i.Use(stdlib.Symbols)

		scriptWithPrelude := wrapGoScript(script)
		if _, err := i.Eval(scriptWithPrelude); err != nil {
			return err
		}
		_, err := i.Eval("__run()")
		return err
	})
	return truncateLog(log, maxOut), runErr
}

func withCapturedOutput(fn func() error) (string, error) {
	var buf bytes.Buffer

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return "", err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return "", err
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		_, _ = io.Copy(&buf, stdoutR)
		_, _ = io.Copy(&buf, stderrR)
	}()

	runErr := fn()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	<-copyDone
	_ = stdoutR.Close()
	_ = stderrR.Close()

	return buf.String(), runErr
}

// wrapGoScript places user code in __run(); leading import blocks stay at package level (yaegi requirement).
func wrapGoScript(script string) string {
	imports, body := splitGoImports(script)
	if imports == "" {
		return fmt.Sprintf("func __run() {\n%s\n}", body)
	}
	return fmt.Sprintf("%s\nfunc __run() {\n%s\n}", imports, body)
}

func splitGoImports(script string) (imports, body string) {
	lines := strings.Split(script, "\n")
	var impLines []string
	var bodyLines []string
	inImportBlock := false
	pastImports := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !pastImports {
			if !inImportBlock && strings.HasPrefix(trimmed, "import ") {
				impLines = append(impLines, line)
				if strings.Contains(trimmed, "(") && !strings.Contains(trimmed, ")") {
					inImportBlock = true
				}
				continue
			}
			if inImportBlock {
				impLines = append(impLines, line)
				if strings.Contains(trimmed, ")") {
					inImportBlock = false
				}
				continue
			}
			if trimmed == "" && len(impLines) > 0 {
				continue
			}
			pastImports = true
		}
		bodyLines = append(bodyLines, line)
	}

	return strings.TrimSpace(strings.Join(impLines, "\n")), strings.TrimSpace(strings.Join(bodyLines, "\n"))
}

func goPath() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".go")
}
