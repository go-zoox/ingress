package service

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reANSI      = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	reLogTime   = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+`)
	reLogTime2  = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+`)
	reLogLev    = regexp.MustCompile(`^(DEBUG|INFO|WARN|ERROR|FATAL)\s+`)
	reHostTag   = regexp.MustCompile(`\[host:\s*([^,\]]+)`)
	reArrowHost     = regexp.MustCompile(`^\S+\s+(\S+)\s+->`)
	reClientHostTarget = regexp.MustCompile(`^(\S+)\s+(\S+)\s+->\s+(\S+)`)
	reUpstreamStatus   = regexp.MustCompile(`upstream_status=(\d+)`)
	reUpstreamTime     = regexp.MustCompile(`upstream_response_time=(\d+)ms`)
	reRealIP           = regexp.MustCompile(`real_ip=([^\s]+)`)
	reRequest          = regexp.MustCompile(`"([A-Z]+)\s+(.+?)\s+HTTP/[^"]+"\s+(\d{3})(?:\s+(\d+(?:\.\d+)?)(ms|s)?)?`)
	reQuotedHTTP       = regexp.MustCompile(`"[A-Z]+\s+.+\s+HTTP/`)
)

// AccessEntry is one parsed ingress access log line.
type AccessEntry struct {
	At                   time.Time
	ClientIP             string
	RealIP               string
	Host                 string
	Target               string
	Method               string
	Path                 string
	Status               int
	DurationMs           float64
	UpstreamStatus       int
	UpstreamDurationMs   float64
	CacheHit             bool
	WAFBlock             bool
}

func parseAccessLine(line string) (AccessEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return AccessEntry{}, false
	}

	// Strip ANSI escape sequences that zoox file transport may include (e.g. colored log levels).
	line = reANSI.ReplaceAllString(line, "")

	var at time.Time
	if m := reLogTime.FindStringSubmatch(line); len(m) == 2 {
		if t, err := time.ParseInLocation("2006/01/02 15:04:05", m[1], time.Local); err == nil {
			at = t
		}
		line = strings.TrimSpace(line[len(m[0]):])
	}

	// Zoox file transport prepends a duplicate timestamp + log level (e.g. "2026/05/24 19:51:04 INFO").
	// Strip the optional second timestamp so the remaining line can be matched by reArrowHost.
	if m := reLogTime2.FindStringSubmatch(line); len(m) == 2 {
		line = strings.TrimSpace(line[len(m[0]):])
	}
	line = reLogLev.ReplaceAllString(line, "")

	clientIP := ""
	host := ""
	target := ""
	if m := reClientHostTarget.FindStringSubmatch(line); len(m) == 4 {
		clientIP = strings.TrimSpace(m[1])
		host = strings.TrimSpace(m[2])
		target = strings.TrimSpace(m[3])
	}
	if host == "" {
		if m := reHostTag.FindStringSubmatch(line); len(m) == 2 {
			host = strings.TrimSpace(m[1])
		} else if m := reArrowHost.FindStringSubmatch(line); len(m) == 2 {
			host = strings.TrimSpace(m[1])
		}
	}
	if host == "" {
		return AccessEntry{}, false
	}

	m := reRequest.FindStringSubmatch(line)
	if len(m) < 4 {
		return AccessEntry{}, false
	}
	status, _ := strconv.Atoi(m[3])
	dur := 0.0
	if m[4] != "" {
		dur, _ = strconv.ParseFloat(m[4], 64)
		if len(m) > 5 && m[5] == "s" {
			dur *= 1000
		}
	}

	upstreamStatus := 0
	if um := reUpstreamStatus.FindStringSubmatch(line); len(um) == 2 {
		upstreamStatus, _ = strconv.Atoi(um[1])
	}
	upstreamDur := 0.0
	if um := reUpstreamTime.FindStringSubmatch(line); len(um) == 2 {
		upstreamDur, _ = strconv.ParseFloat(um[1], 64)
	}
	realIP := ""
	if rm := reRealIP.FindStringSubmatch(line); len(rm) == 2 {
		realIP = strings.TrimSpace(rm[1])
	}

	return AccessEntry{
		At:                 at,
		ClientIP:           clientIP,
		RealIP:             realIP,
		Host:               host,
		Target:             target,
		Method:             m[1],
		Path:               m[2],
		Status:             status,
		DurationMs:         dur,
		UpstreamStatus:     upstreamStatus,
		UpstreamDurationMs: upstreamDur,
		CacheHit:           strings.Contains(line, "cache_hit=1"),
		WAFBlock:           strings.Contains(line, "waf_block=1"),
	}, true
}

// visitorIP returns the best-effort client address for UV counting (real_ip, else client_ip).
func visitorIP(e AccessEntry) string {
	ip := strings.TrimSpace(e.RealIP)
	if ip == "" || ip == "-" {
		ip = strings.TrimSpace(e.ClientIP)
	}
	if ip == "" || ip == "-" {
		return ""
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

// NormalizeLogLine strips ANSI codes and zoox file-transport duplicate timestamps for display.
// Old single-timestamp lines and lines without timestamps are unchanged.
func NormalizeLogLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return line
	}
	line = reANSI.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	m1 := reLogTime.FindStringSubmatch(line)
	if len(m1) != 2 {
		return line
	}
	ts1 := m1[1]
	rest := strings.TrimSpace(line[len(m1[0]):])
	m2 := reLogTime.FindStringSubmatch(rest)
	if len(m2) != 2 {
		return line
	}
	if ts1 != m2[1] {
		return line
	}
	return rest
}

// ParseAccessEntry is the exported version of parseAccessLine for use by handlers.
func ParseAccessEntry(line string) (AccessEntry, bool) {
	return parseAccessLine(line)
}

// ParseIssueCandidate is one access-like log line that failed to parse.
type ParseIssueCandidate struct {
	Line        string
	Reason      string
	Fingerprint string
}

// ParseAccessLogLinesResult summarizes parsing a batch of access log lines.
type ParseAccessLogLinesResult struct {
	Entries      []AccessEntry
	Skipped      int
	IssueSkipped int
	Issues       []ParseIssueCandidate
}

const maxParseIssuesPerScan = 64

// ParseAccessLogLines parses access log lines, skipping noise and collecting parse issues.
func ParseAccessLogLines(lines []string) ParseAccessLogLinesResult {
	out := ParseAccessLogLinesResult{
		Entries: make([]AccessEntry, 0, len(lines)),
	}
	seenIssue := map[string]struct{}{}
	for _, line := range lines {
		e, ok := parseAccessLine(line)
		if ok {
			out.Entries = append(out.Entries, e)
			continue
		}
		out.Skipped++
		if !looksLikeAccessLogLine(line) {
			continue
		}
		out.IssueSkipped++
		if len(out.Issues) >= maxParseIssuesPerScan {
			continue
		}
		fp := fingerprintAccessLogLine(line)
		if fp == "" {
			continue
		}
		if _, dup := seenIssue[fp]; dup {
			continue
		}
		seenIssue[fp] = struct{}{}
		out.Issues = append(out.Issues, ParseIssueCandidate{
			Line:        truncateLine(line, 512),
			Reason:      diagnoseParseFailure(line),
			Fingerprint: fp,
		})
	}
	return out
}

func looksLikeAccessLogLine(line string) bool {
	line = strings.TrimSpace(reANSI.ReplaceAllString(line, ""))
	if line == "" {
		return false
	}
	if strings.Contains(line, " -> ") {
		return true
	}
	if strings.Contains(line, "[host:") {
		return true
	}
	return reQuotedHTTP.MatchString(line)
}

func diagnoseParseFailure(line string) string {
	body := stripAccessLogPrefixes(line)
	if body == "" {
		return "empty_after_prefix"
	}
	if !accessLogBodyHasHost(body) {
		return "missing_host"
	}
	if !reRequest.MatchString(body) {
		return "missing_request"
	}
	return "unknown"
}

func accessLogBodyHasHost(body string) bool {
	if m := reClientHostTarget.FindStringSubmatch(body); len(m) == 4 {
		return strings.TrimSpace(m[2]) != ""
	}
	if m := reHostTag.FindStringSubmatch(body); len(m) == 2 {
		return strings.TrimSpace(m[1]) != ""
	}
	if m := reArrowHost.FindStringSubmatch(body); len(m) == 2 {
		return strings.TrimSpace(m[1]) != ""
	}
	return false
}

// ParseDiagnosis explains why an access-like log line failed to parse.
type ParseDiagnosis struct {
	Reason      string `json:"reason"`
	ReasonLabel string `json:"reason_label"`
	Hint        string `json:"hint"`
	HasHost     bool   `json:"has_host"`
	HasRequest  bool   `json:"has_request"`
	SampleLine  string `json:"sample_line"`
}

// DiagnoseAccessLogLine returns structured parse failure details for admin UI.
func DiagnoseAccessLogLine(line string) ParseDiagnosis {
	body := stripAccessLogPrefixes(line)
	reason := diagnoseParseFailure(line)
	hint := parseReasonHint(reason)
	if reason == "missing_request" && strings.Contains(body, `"`) && strings.Contains(body, "HTTP/") {
		hint = "检测到引号包裹的请求片段，但 \"METHOD /path HTTP/…\" 未能完整匹配；常见原因是 path 中含空格或未转义字符。"
	}
	return ParseDiagnosis{
		Reason:      reason,
		ReasonLabel: parseReasonLabel(reason),
		Hint:        hint,
		HasHost:     accessLogBodyHasHost(body),
		HasRequest:  reRequest.MatchString(body),
		SampleLine:  truncateLine(line, 2048),
	}
}

func parseReasonLabel(reason string) string {
	switch reason {
	case "missing_host":
		return "缺少 host"
	case "missing_request":
		return "缺少 HTTP 请求段"
	case "empty_after_prefix":
		return "去掉前缀后为空"
	default:
		return "格式不兼容"
	}
}

func parseReasonHint(reason string) string {
	switch reason {
	case "missing_host":
		return "需包含 client host -> target，或 [host: …] / host -> target 形式的 host 字段。"
	case "missing_request":
		return "需包含 \"METHOD /path HTTP/1.x\" status duration 段（例如 \"GET / HTTP/1.1\" 200 12ms）。"
	case "empty_after_prefix":
		return "去掉 zoox 时间戳与日志级别前缀后，剩余内容为空。"
	default:
		return "该行看起来像 access 日志，但与 ingress access.log 格式不兼容。"
	}
}

func stripAccessLogPrefixes(line string) string {
	line = strings.TrimSpace(reANSI.ReplaceAllString(line, ""))
	if line == "" {
		return ""
	}
	if m := reLogTime.FindStringSubmatch(line); len(m) == 2 {
		line = strings.TrimSpace(line[len(m[0]):])
	}
	if m := reLogTime2.FindStringSubmatch(line); len(m) == 2 {
		line = strings.TrimSpace(line[len(m[0]):])
	}
	line = strings.TrimSpace(reLogLev.ReplaceAllString(line, ""))
	return line
}

func fingerprintAccessLogLine(line string) string {
	normalized := stripAccessLogPrefixes(line)
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:16])
}

func truncateLine(line string, max int) string {
	line = strings.TrimSpace(line)
	if len(line) <= max {
		return line
	}
	return line[:max] + "…"
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
