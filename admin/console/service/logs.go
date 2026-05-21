package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/go-zoox/ingress/admin/console/config"
)

// LogKind selects access or error log file.
type LogKind string

const (
	LogAccess LogKind = "access"
	LogError  LogKind = "error"
)

// LogQuery filters log lines.
type LogQuery struct {
	Kind   LogKind
	Q      string
	Host   string
	Status string // access only: "", "2", "3", "4", "5"
	Limit  int
}

// Logs reads configured access and error log files.
type Logs struct {
	accessPath string
	errorPath  string
}

func NewLogs(cfg *config.Config) *Logs {
	return &Logs{
		accessPath: strings.TrimSpace(cfg.Ingress.LogPath),
		errorPath:  strings.TrimSpace(cfg.Ingress.ErrorLogPath),
	}
}

// AccessLogPath returns the configured access log file path.
func (l *Logs) AccessLogPath() string {
	if l == nil {
		return ""
	}
	return l.accessPath
}

// ErrorLogPath returns the configured error log file path.
func (l *Logs) ErrorLogPath() string {
	if l == nil {
		return ""
	}
	return l.errorPath
}

func (l *Logs) logPath(kind LogKind) string {
	if kind == LogError {
		return l.errorPath
	}
	return l.accessPath
}

func (l *Logs) Search(q LogQuery) ([]string, error) {
	if q.Kind == "" {
		q.Kind = LogAccess
	}
	path := l.logPath(q.Kind)
	if path == "" {
		return nil, nil
	}
	lines, err := tailLogFile(path, q.Limit)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range lines {
		if matchLogLine(line, q) {
			out = append(out, line)
		}
	}
	return out, nil
}

// Tail reads up to max trailing lines from the access log (for metrics).
func (l *Logs) TailAccess(max int) ([]string, error) {
	path := strings.TrimSpace(l.accessPath)
	if path == "" {
		return nil, nil
	}
	return tailLogFile(path, max)
}

func tailLogFile(path string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open log %s: %w", path, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func matchLogLine(line string, q LogQuery) bool {
	low := strings.ToLower(line)
	if q.Host != "" && !strings.Contains(low, strings.ToLower(q.Host)) {
		return false
	}
	if q.Q != "" && !strings.Contains(low, strings.ToLower(q.Q)) {
		return false
	}
	if q.Kind == LogError || q.Status == "" {
		return true
	}
	if q.Status != "" {
		if !strings.Contains(line, " "+q.Status) && !strings.Contains(line, "\""+q.Status) {
			found := false
			for i := 0; i < len(line)-3; i++ {
				if line[i] == '"' && len(line) > i+4 && line[i+1] >= '0' && line[i+1] <= '9' {
					if string(line[i+1]) == q.Status {
						found = true
						break
					}
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
