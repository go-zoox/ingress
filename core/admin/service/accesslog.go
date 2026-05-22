package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reLogTime = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+`)
	reHostTag = regexp.MustCompile(`\[host:\s*([^,\]]+)`)
	reArrowHost = regexp.MustCompile(`^\S+\s+(\S+)\s+->`)
	reRequest = regexp.MustCompile(`"([A-Z]+)\s+([^\s]+)\s+HTTP/[^"]+"\s+(\d{3})\s+(\d+(?:\.\d+)?)(ms|s)?`)
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

	var at time.Time
	if m := reLogTime.FindStringSubmatch(line); len(m) == 2 {
		if t, err := time.ParseInLocation("2006/01/02 15:04:05", m[1], time.Local); err == nil {
			at = t
		}
		line = strings.TrimSpace(line[len(m[0]):])
	}

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
