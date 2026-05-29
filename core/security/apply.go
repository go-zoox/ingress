package security

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	headerAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	headerAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	headerAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	headerAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	headerAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	headerAccessControlMaxAge           = "Access-Control-Max-Age"
	headerVary                          = "Vary"
	headerOrigin                        = "Origin"
)

// RequestIsHTTPS reports whether the client connection is HTTPS (direct TLS or X-Forwarded-Proto).
func RequestIsHTTPS(req *http.Request) bool {
	if req == nil {
		return false
	}
	if req.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")), "https")
}

// ApplyHeaders sets security response headers on hdr. HSTS is skipped when hstsMode is auto/off and request is HTTP.
func ApplyHeaders(hdr http.Header, prof *Profile, req *http.Request) {
	if prof == nil || !prof.Active {
		return
	}
	for k, v := range prof.Headers {
		if k == headerStrictTransportSecurity {
			if !shouldSendHSTS(prof, req) {
				continue
			}
		}
		if strings.TrimSpace(hdr.Get(k)) != "" {
			continue
		}
		hdr.Set(k, v)
	}
	if prof.CORS != nil {
		applyCORSResponse(hdr, prof.CORS, req)
	}
}

func shouldSendHSTS(prof *Profile, req *http.Request) bool {
	if prof == nil {
		return false
	}
	if _, ok := prof.Headers[headerStrictTransportSecurity]; !ok {
		return false
	}
	// compileOne only adds HSTS when mode != off; auto defers to HTTPS detection here.
	return RequestIsHTTPS(req)
}

func applyCORSResponse(hdr http.Header, cors *CORSProfile, req *http.Request) {
	if cors == nil || req == nil {
		return
	}
	origin := strings.TrimSpace(req.Header.Get(headerOrigin))
	if origin == "" {
		return
	}
	allowed, ok := matchOrigin(origin, cors.AllowOrigins)
	if !ok {
		return
	}
	hdr.Set(headerAccessControlAllowOrigin, allowed)
	if cors.AllowCredentials {
		hdr.Set(headerAccessControlAllowCredentials, "true")
	}
	if len(cors.ExposeHeaders) > 0 {
		hdr.Set(headerAccessControlExposeHeaders, strings.Join(cors.ExposeHeaders, ", "))
	}
	hdr.Add(headerVary, headerOrigin)
}

// HandlePreflight responds to OPTIONS preflight when CORS is configured. Returns true when handled.
func HandlePreflight(w http.ResponseWriter, req *http.Request, prof *Profile) bool {
	if prof == nil || prof.CORS == nil || req == nil {
		return false
	}
	if req.Method != http.MethodOptions {
		return false
	}
	origin := strings.TrimSpace(req.Header.Get(headerOrigin))
	if origin == "" {
		return false
	}
	acrm := strings.TrimSpace(req.Header.Get("Access-Control-Request-Method"))
	if acrm == "" {
		return false
	}
	allowed, ok := matchOrigin(origin, prof.CORS.AllowOrigins)
	if !ok {
		return false
	}
	hdr := w.Header()
	hdr.Set(headerAccessControlAllowOrigin, allowed)
	hdr.Set(headerAccessControlAllowMethods, strings.Join(prof.CORS.AllowMethods, ", "))
	hdr.Set(headerAccessControlAllowHeaders, strings.Join(prof.CORS.AllowHeaders, ", "))
	if prof.CORS.AllowCredentials {
		hdr.Set(headerAccessControlAllowCredentials, "true")
	}
	if prof.CORS.MaxAge > 0 {
		hdr.Set(headerAccessControlMaxAge, formatMaxAge(prof.CORS.MaxAge))
	}
	hdr.Add(headerVary, headerOrigin)
	ApplyHeaders(hdr, prof, req)
	w.WriteHeader(http.StatusNoContent)
	return true
}

func matchOrigin(origin string, allowed []string) (string, bool) {
	for _, o := range allowed {
		if o == "*" {
			return "*", true
		}
		if strings.EqualFold(o, origin) {
			return origin, true
		}
	}
	return "", false
}

func formatMaxAge(sec int64) string {
	return strconv.FormatInt(sec, 10)
}
