package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/go-zoox/ingress/core/admin/config"
)

// LogKind selects access or error log file.
type LogKind string

const (
	LogAccess LogKind = "access"
	LogError  LogKind = "error"
)

// LogQuery filters log lines.
type LogQuery struct {
	Kind      LogKind
	Q         string
	Host      string
	Status    string // access only: "", "2", "3", "4", "5"
	CacheHit  string // access only: "", "0", "1"
	WAFBlock  string // access only: "", "0", "1"
	Limit     int
	Offset    int64 // byte offset for incremental tail; 0 = snapshot tail
}

// LogResult is the logs API payload.
type LogResult struct {
	Lines  []string `json:"lines"`
	Count  int      `json:"count"`
	Offset int64    `json:"offset"`
}

// Logs reads configured access and error log files.
type Logs struct {
	accessPath string
	errorPath  string
}

func NewLogs(cfg *config.Config) *Logs {
	return &Logs{
		accessPath: strings.TrimSpace(cfg.AccessLogPath),
		errorPath:  strings.TrimSpace(cfg.ErrorLogPath),
	}
}

func (l *Logs) AccessLogPath() string {
	if l == nil {
		return ""
	}
	return l.accessPath
}

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

func (l *Logs) Search(q LogQuery) (LogResult, error) {
	if q.Kind == "" {
		q.Kind = LogAccess
	}
	path := l.logPath(q.Kind)
	if path == "" {
		return LogResult{}, nil
	}
	if q.Offset > 0 {
		return tailSinceOffset(path, q)
	}
	lines, err := tailLogFile(path, q.Limit)
	if err != nil {
		return LogResult{}, err
	}
	out := filterLines(lines, q)
	size, _ := fileSize(path)
	return LogResult{Lines: out, Count: len(out), Offset: size}, nil
}

func (l *Logs) TailAccess(max int) ([]string, error) {
	path := strings.TrimSpace(l.accessPath)
	if path == "" {
		return nil, nil
	}
	return tailLogFile(path, max)
}

func tailSinceOffset(path string, q LogQuery) (LogResult, error) {
	size, err := fileSize(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LogResult{}, nil
		}
		return LogResult{}, err
	}
	if q.Offset >= size {
		return LogResult{Offset: size}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LogResult{}, nil
		}
		return LogResult{}, fmt.Errorf("open log %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.Seek(q.Offset, io.SeekStart); err != nil {
		return LogResult{}, err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return LogResult{}, err
	}
	raw := strings.Split(string(data), "\n")
	if len(raw) > 0 && raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}
	// First fragment after a byte offset may be a partial line; drop it.
	if q.Offset > 0 && len(raw) > 0 {
		raw = raw[1:]
	}
	out := filterLines(raw, q)
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[len(out)-q.Limit:]
	}
	return LogResult{Lines: out, Count: len(out), Offset: size}, nil
}

func filterLines(lines []string, q LogQuery) []string {
	var out []string
	for _, line := range lines {
		if matchLogLine(line, q) {
			out = append(out, line)
		}
	}
	return out
}

func fileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
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

// DistinctHosts extracts unique host values from the access log (last 5000 lines).
func (l *Logs) DistinctHosts() ([]string, error) {
	path := strings.TrimSpace(l.accessPath)
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open log %s: %w", path, err)
	}
	defer f.Close()

	// tail last 5000 lines
	var allLines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		allLines = append(allLines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(allLines) > 5000 {
		allLines = allLines[len(allLines)-5000:]
	}

	seen := make(map[string]struct{})
	for _, line := range allLines {
		if entry, ok := parseAccessLine(line); ok && entry.Host != "" {
			seen[entry.Host] = struct{}{}
		}
	}

	hosts := make([]string, 0, len(seen))
	for h := range seen {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	return hosts, nil
}

func matchLogLine(line string, q LogQuery) bool {
	low := strings.ToLower(line)
	if q.Host != "" && !strings.Contains(low, strings.ToLower(q.Host)) {
		return false
	}
	if q.Q != "" && !strings.Contains(low, strings.ToLower(q.Q)) {
		return false
	}
	if q.Kind == LogAccess && q.CacheHit != "" {
		want := "cache_hit=" + q.CacheHit
		if !strings.Contains(line, want) {
			return false
		}
	}
	if q.Kind == LogAccess && q.WAFBlock != "" {
		want := "waf_block=" + q.WAFBlock
		if !strings.Contains(line, want) {
			return false
		}
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
