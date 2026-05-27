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
			Pattern: `(?is)(<\s*script\b|javascript:\s*[a-z]|\bon(?:click|load|error|focus|blur|change|submit|mouse\w*|key\w*|touch\w*|pointer\w*|scroll|dblclick|drag\w*|drop|input|reset|select|wheel|copy|cut|paste|abort|contextmenu|message|unload|beforeunload)\s*=)`,
			Targets: []string{TargetURI},
		},
		{
			ID:      "builtin:rce-probes",
			Name:    "Command injection probes",
			Type:    PatternTypeRegex,
			Pattern: `(?i)(\||;|&&|\$\(|` + "`" + `)\s*(cat|curl|wget|bash|/bin/sh|cmd\.exe|powershell)\b`,
			Targets: []string{TargetURI},
		},
		{
			ID:      "builtin:jndi-lookup",
			Name:    "JNDI lookup injection",
			Type:    PatternTypeRegex,
			Pattern: `(?i)\$\{[^}]*jndi:`,
			Targets: []string{TargetURI, TargetHeaders},
		},
		{
			ID:      "builtin:sensitive-files",
			Name:    "Sensitive file and admin path probes",
			Type:    PatternTypeRegex,
			Pattern: `(?i)(/\.env\b|/\.git/|/wp-admin\b|/phpmyadmin\b|/web\.config\b|/id_rsa\b)`,
			Targets: []string{TargetPath},
		},
		{
			ID:      "builtin:ssrf-probes",
			Name:    "SSRF and metadata endpoint probes",
			Type:    PatternTypeRegex,
			Pattern: `(?i)(169\.254\.169\.254|metadata\.google|file://|gopher://)`,
			Targets: []string{TargetURI},
		},
		{
			ID:      "builtin:scanner-ua",
			Name:    "Known scanner User-Agent",
			Type:    PatternTypeRegex,
			Pattern: `(?i)(sqlmap|nikto|nmap|masscan|acunetix|nessus|dirbuster|gobuster)`,
			Targets: []string{"header:User-Agent"},
		},
		{
			ID:      "builtin:crlf-injection",
			Name:    "CRLF / response-splitting probes",
			Type:    PatternTypeRegex,
			Pattern: `(?i)(%0d%0a|%0a%0d|\r\n)(content-length|set-cookie|location):`,
			Targets: []string{TargetURI, TargetHeaders},
		},
		{
			ID:      "builtin:php-ssti",
			Name:    "PHP / template injection probes",
			Type:    PatternTypeRegex,
			Pattern: `(?i)(eval\s*\(|base64_decode\s*\(|php://|expect://)`,
			Targets: []string{TargetURI},
		},
	}
}
