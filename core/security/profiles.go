package security

import (
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

const (
	ProfileStrict     = "strict"
	ProfileAPI        = "api"
	ProfileEmbeddable = "embeddable"
	ProfileOff        = "off"

	HSTSAuto = "auto"
	HSTSOn   = "on"
	HSTSOff  = "off"

	FrameInherit     = "inherit"
	FrameDeny        = "deny"
	FrameSameOrigin  = "sameorigin"
	FrameOff         = "off"
	ReferrerOff      = "off"
	CSPOff           = "off"
)

type profileDefaults struct {
	hsts               string
	frame              string
	contentTypeOptions bool
	referrerPolicy     string
	csp                string
	corsEnabled        bool
}

var profileTable = map[string]profileDefaults{
	ProfileStrict: {
		hsts:               HSTSAuto,
		frame:              FrameDeny,
		contentTypeOptions: true,
		referrerPolicy:     "strict-origin-when-cross-origin",
		csp:                "default-src 'self'; frame-ancestors 'none'",
		corsEnabled:        false,
	},
	ProfileAPI: {
		hsts:               HSTSAuto,
		frame:              FrameDeny,
		contentTypeOptions: true,
		referrerPolicy:     "strict-origin-when-cross-origin",
		csp:                "default-src 'none'; frame-ancestors 'none'",
		corsEnabled:        true,
	},
	ProfileEmbeddable: {
		hsts:               HSTSAuto,
		frame:              FrameSameOrigin,
		contentTypeOptions: true,
		referrerPolicy:     "strict-origin-when-cross-origin",
		csp:                "frame-ancestors 'self'",
		corsEnabled:        false,
	},
}

func normalizeProfile(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func mergeSecurity(global, perRule rule.Security) rule.Security {
	if !hasSecurityConfig(perRule) {
		return global
	}
	out := global
	if p := normalizeProfile(perRule.Profile); p != "" {
		out.Profile = p
	}
	if h := strings.ToLower(strings.TrimSpace(perRule.HSTS)); h != "" {
		out.HSTS = h
	}
	if f := strings.ToLower(strings.TrimSpace(perRule.Frame)); f != "" {
		out.Frame = f
	}
	if perRule.ContentTypeOptions != nil {
		out.ContentTypeOptions = perRule.ContentTypeOptions
	}
	if rp := strings.TrimSpace(perRule.ReferrerPolicy); rp != "" {
		out.ReferrerPolicy = rp
	}
	if csp := strings.TrimSpace(perRule.CSP); csp != "" {
		out.CSP = csp
	}
	out.CORS = mergeCORS(global.CORS, perRule.CORS)
	return out
}

func mergeCORS(base, override rule.CORS) rule.CORS {
	out := base
	if override.Enabled != nil {
		out.Enabled = override.Enabled
	}
	if len(override.Origins) > 0 {
		out.Origins = append([]string(nil), override.Origins...)
	}
	if len(override.Methods) > 0 {
		out.Methods = append([]string(nil), override.Methods...)
	}
	if len(override.Headers) > 0 {
		out.Headers = append([]string(nil), override.Headers...)
	}
	if len(override.ExposeHeaders) > 0 {
		out.ExposeHeaders = append([]string(nil), override.ExposeHeaders...)
	}
	if override.Credentials != nil {
		out.Credentials = override.Credentials
	}
	if override.MaxAge > 0 {
		out.MaxAge = override.MaxAge
	}
	return out
}

func hasSecurityConfig(s rule.Security) bool {
	if normalizeProfile(s.Profile) != "" {
		return true
	}
	if strings.TrimSpace(s.HSTS) != "" || strings.TrimSpace(s.Frame) != "" {
		return true
	}
	if s.ContentTypeOptions != nil {
		return true
	}
	if strings.TrimSpace(s.ReferrerPolicy) != "" || strings.TrimSpace(s.CSP) != "" {
		return true
	}
	return hasCORSConfig(s.CORS)
}

func hasCORSConfig(c rule.CORS) bool {
	if c.Enabled != nil {
		return true
	}
	if len(c.Origins) > 0 || len(c.Methods) > 0 || len(c.Headers) > 0 || len(c.ExposeHeaders) > 0 {
		return true
	}
	if c.Credentials != nil || c.MaxAge > 0 {
		return true
	}
	return false
}
