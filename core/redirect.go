package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/zoox"
)

const (
	redirectDurationPermanent = "permanent"
	redirectDurationTemporary = "temporary"
)

// RedirectBehavior is the resolved redirect semantics (scheme A).
type RedirectBehavior struct {
	Permanent       bool
	PreserveRequest bool
}

func effectiveRedirectBehavior(duration string, preserveRequest, permanent, withOriginMethodAndBody bool) (RedirectBehavior, error) {
	dur := strings.ToLower(strings.TrimSpace(duration))
	b := RedirectBehavior{
		Permanent:       permanent,
		PreserveRequest: preserveRequest || withOriginMethodAndBody,
	}

	switch dur {
	case "":
	case redirectDurationPermanent:
		b.Permanent = true
	case redirectDurationTemporary:
		b.Permanent = false
	default:
		return b, fmt.Errorf("redirect.duration must be %q or %q", redirectDurationPermanent, redirectDurationTemporary)
	}

	if dur == redirectDurationTemporary && permanent {
		return b, fmt.Errorf("redirect.duration %q conflicts with redirect.permanent: true", redirectDurationTemporary)
	}

	return b, nil
}

func effectiveRuleRedirectBehavior(r rule.Redirect) (RedirectBehavior, error) {
	return effectiveRedirectBehavior(r.Duration, r.PreserveRequest, r.Permanent, r.WithOriginMethodAndBody)
}

func effectiveHTTPRedirectBehavior(r RedirectFromHTTP) (RedirectBehavior, error) {
	return effectiveRedirectBehavior(r.Duration, r.PreserveRequest, r.Permanent, r.WithOriginMethodAndBody)
}

// validateRuleRedirect checks one backend.redirect block. pathLevel marks a paths[].backend.redirect:
// preserve_path only exists there, because host-level and fallback redirects always keep the request
// path and query for whole-site moves.
func validateRuleRedirect(r rule.Redirect, loc string, pathLevel bool) error {
	if strings.TrimSpace(r.URL) == "" {
		return nil
	}
	if r.PreservePath && !pathLevel {
		return fmt.Errorf("%s: preserve_path is only supported on paths[].backend.redirect (host-level redirects always keep the request path and query)", loc)
	}
	if _, err := effectiveRuleRedirectBehavior(r); err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	return nil
}

func validateRedirectFromHTTP(r RedirectFromHTTP) error {
	if !r.Enabled {
		return nil
	}
	if _, err := effectiveHTTPRedirectBehavior(r); err != nil {
		return fmt.Errorf("https.redirect_from_http: %w", err)
	}
	return nil
}

func redirectStatusCode(b RedirectBehavior) int {
	if b.PreserveRequest {
		if b.Permanent {
			return http.StatusPermanentRedirect
		}
		return http.StatusTemporaryRedirect
	}
	if b.Permanent {
		return http.StatusMovedPermanently
	}
	return http.StatusFound
}

func applyRedirectBehavior(ctx *zoox.Context, url string, b RedirectBehavior) {
	if b.PreserveRequest {
		if b.Permanent {
			ctx.RedirectPermanentWithOriginMethodAndBody(url)
		} else {
			ctx.RedirectTemporaryWithOriginMethodAndBody(url)
		}
		return
	}
	if b.Permanent {
		ctx.RedirectPermanent(url)
	} else {
		ctx.RedirectTemporary(url)
	}
}
