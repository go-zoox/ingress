// Package peerip carries the real TCP peer address of a request.
//
// zoox's default realip middleware rewrites http.Request.RemoteAddr from
// True-Client-IP / X-Real-IP / X-Forwarded-For, which means RemoteAddr can be
// client-controlled. Security decisions (WAF IP allowlist) must not rely on it,
// so ingress captures the TCP peer before that rewrite and exposes it through
// the request context.
package peerip

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type contextKey struct{}

// With stores the TCP peer IP (without port) in ctx.
func With(ctx context.Context, ip string) context.Context {
	if ctx == nil || ip == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, ip)
}

// FromContext returns the TCP peer IP captured by ingress, or "" when absent.
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(contextKey{}).(string)
	return v
}

// FromRequest returns the captured TCP peer IP for r, or "" when unavailable.
func FromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	return FromContext(r.Context())
}

// RemoteAddrIP parses an IP from a host:port or bare-IP string.
func RemoteAddrIP(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return ""
	}
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return ""
}
