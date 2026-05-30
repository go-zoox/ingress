package jobs

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultScriptShell = "sh"

// UnmarshalYAML accepts legacy `shell: true|false` booleans alongside string shell names.
func (p *JobParams) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}
	if shell, ok := m["shell"]; ok {
		switch v := shell.(type) {
		case bool:
			if v {
				m["shell"] = DefaultScriptShell
			} else {
				delete(m, "shell")
			}
		case string:
			m["shell"] = v
		}
	}
	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	type plain JobParams
	return yaml.Unmarshal(b, (*plain)(p))
}

// ScriptContent returns the script body, migrating legacy command/args when needed.
func ScriptContent(p JobParams) string {
	if strings.TrimSpace(p.Script) != "" {
		return p.Script
	}
	cmd := strings.TrimSpace(p.Command)
	if cmd == "" {
		return ""
	}
	if len(p.Args) > 0 {
		return cmd + " " + strings.Join(p.Args, " ")
	}
	return cmd
}

// ScriptShellName returns the shell interpreter when engine is shell (default sh).
func ScriptShellName(p JobParams) string {
	if NormalizeScriptEngine(p.Engine) != ScriptEngineShell {
		return ""
	}
	shell := strings.TrimSpace(p.Shell)
	if shell != "" {
		return shell
	}
	return DefaultScriptShell
}

// ResolveShellExecutable maps a shell name or path to an executable path.
func ResolveShellExecutable(shell string) (string, error) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		shell = DefaultScriptShell
	}
	switch shell {
	case "sh":
		return "/bin/sh", nil
	case "bash":
		return "/bin/bash", nil
	case "zsh":
		return "/bin/zsh", nil
	case "dash":
		return "/bin/dash", nil
	default:
		if strings.Contains(shell, "/") {
			return shell, nil
		}
		return "/bin/" + shell, nil
	}
}

// PrepareScriptParams normalizes script fields on a params value for storage and execution.
func PrepareScriptParams(p *JobParams) error {
	if p == nil {
		return fmt.Errorf("script params are nil")
	}
	content := strings.TrimSpace(ScriptContent(*p))
	if content == "" {
		return fmt.Errorf("script requires params.script")
	}
	p.Script = ScriptContent(*p)
	p.Engine = NormalizeScriptEngine(p.Engine)
	if p.Engine == ScriptEngineShell {
		p.Shell = ScriptShellName(*p)
	} else {
		p.Shell = ""
	}
	p.Command = ""
	p.Args = nil
	return nil
}

// ValidateScriptEngineParams checks engine-specific param rules before normalization.
func ValidateScriptEngineParams(p *JobParams) error {
	if p == nil {
		return fmt.Errorf("script params are nil")
	}
	engine := NormalizeScriptEngine(p.Engine)
	if engine != ScriptEngineShell && strings.TrimSpace(p.Shell) != "" {
		return fmt.Errorf("params.shell is only allowed when engine is shell (got engine=%q)", engine)
	}
	switch engine {
	case ScriptEngineShell, ScriptEngineJavaScript, ScriptEngineGo:
		return nil
	default:
		return fmt.Errorf("unsupported script engine %q (use shell, javascript, or go)", p.Engine)
	}
}
