package security

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

const (
	headerStrictTransportSecurity = "Strict-Transport-Security"
	headerXFrameOptions           = "X-Frame-Options"
	headerXContentTypeOptions     = "X-Content-Type-Options"
	headerReferrerPolicy          = "Referrer-Policy"
	headerContentSecurityPolicy   = "Content-Security-Policy"
)

// Profile is compiled security headers for one effective security config.
type Profile struct {
	Active  bool
	Headers map[string]string
	CORS    *CORSProfile
}

// CORSProfile is compiled CORS settings.
type CORSProfile struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int64
}

// Ingress holds compiled global, per-rule, and per-path security profiles.
type Ingress struct {
	Global *Profile
	ByRule []*Profile
	ByPath [][]*Profile
}

// Compile builds security profiles from config.
func Compile(global rule.Security, rules []rule.Rule) (*Ingress, error) {
	out := &Ingress{
		ByRule: make([]*Profile, len(rules)),
		ByPath: make([][]*Profile, len(rules)),
	}
	gp, err := compileOne(global, "security")
	if err != nil {
		return nil, err
	}
	out.Global = gp

	for i := range rules {
		merged := mergeSecurity(global, rules[i].Security)
		p, err := compileOne(merged, fmt.Sprintf("rules[%d].security", i))
		if err != nil {
			return nil, err
		}
		out.ByRule[i] = p

		paths := rules[i].Paths
		out.ByPath[i] = make([]*Profile, len(paths))
		for j := range paths {
			pathMerged := mergeSecurity(merged, paths[j].Security)
			pp, err := compileOne(pathMerged, fmt.Sprintf("rules[%d].paths[%d].security", i, j))
			if err != nil {
				return nil, err
			}
			out.ByPath[i][j] = pp
		}
	}
	return out, nil
}

func compileOne(cfg rule.Security, label string) (*Profile, error) {
	profileName := normalizeProfile(cfg.Profile)
	var def profileDefaults
	switch profileName {
	case "", ProfileOff:
		if !hasSecurityConfig(cfg) {
			return &Profile{Active: false, Headers: map[string]string{}}, nil
		}
		def = profileDefaults{hsts: HSTSOff, frame: FrameOff, contentTypeOptions: false}
	case ProfileStrict, ProfileAPI, ProfileEmbeddable:
		var ok bool
		def, ok = profileTable[profileName]
		if !ok {
			return nil, fmt.Errorf("%s.profile: unsupported %q (use strict, api, embeddable, or off)", label, cfg.Profile)
		}
	default:
		return nil, fmt.Errorf("%s.profile: unsupported %q (use strict, api, embeddable, or off)", label, cfg.Profile)
	}

	hstsMode := strings.ToLower(strings.TrimSpace(cfg.HSTS))
	if hstsMode == "" {
		hstsMode = def.hsts
	}
	if err := validateEnum(label+".hsts", hstsMode, HSTSAuto, HSTSOn, HSTSOff); err != nil {
		return nil, err
	}

	frameMode := strings.ToLower(strings.TrimSpace(cfg.Frame))
	if frameMode == "" || frameMode == FrameInherit {
		frameMode = def.frame
	}
	if err := validateEnum(label+".frame", frameMode, FrameDeny, FrameSameOrigin, FrameOff); err != nil {
		return nil, err
	}

	cto := def.contentTypeOptions
	if cfg.ContentTypeOptions != nil {
		cto = *cfg.ContentTypeOptions
	}

	referrer := strings.TrimSpace(cfg.ReferrerPolicy)
	if referrer == "" {
		referrer = def.referrerPolicy
	}

	csp := strings.TrimSpace(cfg.CSP)
	if csp == "" {
		csp = def.csp
	}

	headers := map[string]string{}
	if hstsMode != HSTSOff {
		headers[headerStrictTransportSecurity] = "max-age=31536000; includeSubDomains"
	}
	switch frameMode {
	case FrameDeny:
		headers[headerXFrameOptions] = "DENY"
	case FrameSameOrigin:
		headers[headerXFrameOptions] = "SAMEORIGIN"
	}
	if cto {
		headers[headerXContentTypeOptions] = "nosniff"
	}
	if referrer != "" && !strings.EqualFold(referrer, ReferrerOff) {
		headers[headerReferrerPolicy] = referrer
	}
	if csp != "" && !strings.EqualFold(csp, CSPOff) {
		headers[headerContentSecurityPolicy] = csp
	}

	corsProf, err := compileCORS(cfg.CORS, def.corsEnabled, label+".cors")
	if err != nil {
		return nil, err
	}

	active := len(headers) > 0 || corsProf != nil
	return &Profile{Active: active, Headers: headers, CORS: corsProf}, nil
}

func compileCORS(cfg rule.CORS, profileDefault bool, label string) (*CORSProfile, error) {
	enabled := profileDefault
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}
	if !enabled {
		return nil, nil
	}
	origins := trimNonEmpty(cfg.Origins)
	if len(origins) == 0 {
		return nil, fmt.Errorf("%s: cors enabled but origins is empty", label)
	}
	methods := trimNonEmpty(cfg.Methods)
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	headers := trimNonEmpty(cfg.Headers)
	if len(headers) == 0 {
		headers = []string{"Authorization", "Content-Type", "Accept", "X-Requested-With"}
	}
	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}
	creds := false
	if cfg.Credentials != nil {
		creds = *cfg.Credentials
	}
	for _, o := range origins {
		if o == "*" && creds {
			return nil, fmt.Errorf("%s: credentials cannot be true when origins contains *", label)
		}
	}
	return &CORSProfile{
		AllowOrigins:     origins,
		AllowMethods:     methods,
		AllowHeaders:     headers,
		ExposeHeaders:    trimNonEmpty(cfg.ExposeHeaders),
		AllowCredentials: creds,
		MaxAge:           maxAge,
	}, nil
}

func validateEnum(label, value string, allowed ...string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s: unsupported %q", label, value)
}

func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
