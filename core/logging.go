package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	zcfg "github.com/go-zoox/zoox/config"
)

const (
	DefaultLogDir        = "/var/log/ingress"
	DefaultAccessLogPath = DefaultLogDir + "/access.log"
	DefaultErrorLogPath  = DefaultLogDir + "/error.log"
)

// Logging is ingress logging config (YAML key `logging`).
// Zoox always includes console; file sinks come from Transports or logging.enable defaults.
type Logging struct {
	// Enable turns on default file logging (console + /var/log/ingress/*.log) when Transports is empty.
	Enable *bool `config:"enable"`
	Level  string `config:"level"`
	//
	Middleware zcfg.Middleware `config:"middleware"`
	Transports []zcfg.Transport `config:"transports"`
}

func DefaultFileTransport() []zcfg.Transport {
	return []zcfg.Transport{{
		Type: "file",
		Path: DefaultAccessLogPath,
		Levels: map[string]string{
			"error": DefaultErrorLogPath,
		},
	}}
}

func (l *Logging) FileLoggingEnabled() bool {
	if l == nil || l.Enable == nil {
		return false
	}
	return *l.Enable
}

func (l *Logging) Configured() bool {
	if l == nil {
		return false
	}
	if l.FileLoggingEnabled() {
		return true
	}
	if strings.TrimSpace(l.Level) != "" {
		return true
	}
	return len(l.Transports) > 0
}

func (l *Logging) Zoox() zcfg.Logger {
	if l == nil {
		return zcfg.Logger{}
	}
	return zcfg.Logger{
		Level:      l.Level,
		Middleware: l.Middleware,
		Transports: l.Transports,
	}
}

// Normalize applies logging.enable defaults, clears file sinks when disabled, and creates log directories.
func (l *Logging) Normalize() error {
	if l == nil {
		return nil
	}
	l.applyFileDefaults()
	paths := l.logFilePaths()
	if len(paths) == 0 {
		return nil
	}
	return EnsureLogDirectories(paths...)
}

func (l *Logging) applyFileDefaults() {
	if l == nil {
		return
	}
	if l.Enable != nil && !*l.Enable {
		l.Transports = nil
		return
	}
	if l.FileLoggingEnabled() && len(l.Transports) == 0 {
		l.Transports = DefaultFileTransport()
	}
}

// FileLogPaths returns access and error log paths from file transports (after Normalize).
func (l *Logging) FileLogPaths() (access, errorLog string) {
	if l == nil {
		return "", ""
	}
	for _, t := range l.Transports {
		typ := strings.ToLower(strings.TrimSpace(t.Type))
		if typ != "" && typ != "file" {
			continue
		}
		if strings.TrimSpace(t.Path) != "" {
			access = t.Path
		}
		if p := strings.TrimSpace(t.Levels["error"]); p != "" {
			errorLog = p
		}
	}
	return access, errorLog
}

func (l *Logging) logFilePaths() []string {
	var out []string
	for _, t := range l.Transports {
		typ := strings.ToLower(strings.TrimSpace(t.Type))
		if typ != "" && typ != "file" {
			continue
		}
		if p := strings.TrimSpace(t.Path); p != "" {
			out = append(out, p)
		}
		for _, p := range t.Levels {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// EnsureLogDirectories creates parent directories for log file paths.
func EnsureLogDirectories(paths ...string) error {
	seen := map[string]struct{}{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		dir := filepath.Dir(p)
		if dir == "" || dir == "." {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create log directory %s: %w", dir, err)
		}
	}
	return nil
}
