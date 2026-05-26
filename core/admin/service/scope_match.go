package service

import "strings"

// MatchPathForScope matches requestPath against scopePath using the given mode.
//
// Modes:
// - "exact": requestPath == scopePath
// - "prefix" (default): scopePath is a path prefix boundary.
//   Example: scopePath="/api/user" matches "/api/user" and "/api/user/123".
func MatchPathForScope(requestPath, scopePath, mode string) bool {
	scopePath = strings.TrimSpace(scopePath)
	requestPath = strings.TrimSpace(requestPath)
	if scopePath == "" {
		return true
	}

	// Best-effort normalization for scopePath.
	if !strings.HasPrefix(scopePath, "/") {
		scopePath = "/" + scopePath
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "prefix"
	}

	switch m {
	case "exact":
		return requestPath == scopePath
	case "prefix":
		// Trim trailing slashes for stable matching.
		sp := strings.TrimRight(scopePath, "/")
		rp := strings.TrimRight(requestPath, "/")
		if sp == "" {
			sp = "/"
		}
		if rp == "" {
			rp = "/"
		}
		if sp == "/" {
			return true
		}
		if rp == sp {
			return true
		}
		return strings.HasPrefix(rp, sp+"/")
	default:
		// Unknown mode: treat as prefix.
		sp := strings.TrimRight(scopePath, "/")
		rp := strings.TrimRight(requestPath, "/")
		if sp == "" {
			sp = "/"
		}
		if rp == "" {
			rp = "/"
		}
		if sp == "/" {
			return true
		}
		if rp == sp {
			return true
		}
		return strings.HasPrefix(rp, sp+"/")
	}
}

