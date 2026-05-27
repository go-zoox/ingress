package waf

import "github.com/go-zoox/ingress/core/rule"

// StarterRules returns embedded starter signatures (IDs stable for logging / overrides).
func StarterRules() []rule.WAFRule {
	return []rule.WAFRule{
		{
			ID:      "builtin:sqli-common",
			Name:    "SQL injection probes (query/url)",
			Type:    PatternTypeRegex,
			Pattern: `(?is)(union\s+select\b|sleep\s*\(|benchmark\s*\(|;\s*(drop|truncate|alter)\s+table\b)`,
			Targets: []string{TargetURI},
		},
		{
			ID:      "builtin:path-traversal",
			Name:    "Path traversal in request line",
			Type:    PatternTypeRegex,
			Pattern: `(?:\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\\\|etc/passwd\b)`,
			Targets: []string{TargetPath},
		},
		{
			ID:      "builtin:xss-lite",
			Name:    "Reflected scripting probes (lite)",
			Type:    PatternTypeRegex,
			Pattern: `(?is)(<\s*script\b|javascript:\s*|\bon[a-z]+\s*=)`,
			Targets: []string{TargetURI},
		},
	}
}
