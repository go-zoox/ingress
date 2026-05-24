package service

import (
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
	reArrowHost = regexp.MustCompile(`^\S+\s+(\S+)\s+->`)
	reRequest   = regexp.MustCompile(`"([A-Z]+)\s+([^\s]+)\s+HTTP/[^"]+"\s+(\d{3})\s+(\d+(?:\.\d+)?)(ms|s)?`)
)

// AccessEntry is one parsed ingress access log line.
type AccessEntry struct {
	At         time.Time
	Host       string
	Method     string
	Path       string
	Status     int
	DurationMs float64
	CacheHit   bool
	WAFBlock   bool
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

	host := ""
	if m := reHostTag.FindStringSubmatch(line); len(m) == 2 {
		host = strings.TrimSpace(m[1])
	} else if m := reArrowHost.FindStringSubmatch(line); len(m) == 2 {
		host = strings.TrimSpace(m[1])
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

	return AccessEntry{
		At:         at,
		Host:       host,
		Method:     m[1],
		Path:       m[2],
		Status:     status,
		DurationMs: dur,
		CacheHit:   strings.Contains(line, "cache_hit=1"),
		WAFBlock:   strings.Contains(line, "waf_block=1"),
	}, true
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
