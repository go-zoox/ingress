package waf

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type hostMatcher struct {
	exact string
	re    *regexp.Regexp
}

func hostLooksLikeRegexp(host string) bool {
	for _, r := range host {
		switch r {
		case '(', ')', '[', ']', '^', '$', '|', '+', '?', '\\':
			return true
		}
	}
	return false
}

func effectiveHostMatchKind(host string) string {
	if hostLooksLikeRegexp(host) {
		return "regex"
	}
	if strings.Contains(host, "*") {
		return "wildcard"
	}
	return "exact"
}

func wildCardHostToRegexp(pattern string) string {
	components := strings.Split(pattern, "*")
	if len(components) == 1 {
		return "^" + regexp.QuoteMeta(pattern) + "$"
	}
	var result strings.Builder
	for i, literal := range components {
		if i > 0 {
			result.WriteString(".*")
		}
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return "^" + result.String() + "$"
}

func compileHostPattern(raw string) (hostMatcher, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return hostMatcher{}, fmt.Errorf("empty host pattern")
	}
	switch effectiveHostMatchKind(raw) {
	case "exact":
		return hostMatcher{exact: raw}, nil
	default:
		pat := raw
		if effectiveHostMatchKind(raw) == "wildcard" {
			pat = wildCardHostToRegexp(raw)
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return hostMatcher{}, err
		}
		return hostMatcher{re: re}, nil
	}
}

func normalizeRequestHost(hostname string) string {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return hostname
	}
	if host, _, err := net.SplitHostPort(hostname); err == nil {
		return host
	}
	return hostname
}

func hostMatchesAllowList(hostname string, matchers []hostMatcher) bool {
	host := normalizeRequestHost(hostname)
	if host == "" || len(matchers) == 0 {
		return false
	}
	for _, m := range matchers {
		if m.exact != "" && m.exact == host {
			return true
		}
		if m.re != nil && m.re.MatchString(host) {
			return true
		}
	}
	return false
}
