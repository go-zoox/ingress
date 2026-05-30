package core

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type accessLogMeta struct {
	CacheHit               bool
	WAFBlock               bool
	RateLimitBlock         bool
	MaintenanceBlock       bool
	UpstreamStatus         int
	UpstreamResponseLength int64
	UpstreamResponseTime   time.Duration
}

func formatAccessLog(req *http.Request, host, target, method, path, proto string, status int, dur time.Duration, meta accessLogMeta) string {
	clientIP := accessLogClientIP(req)
	if meta.UpstreamStatus == 0 {
		meta.UpstreamStatus = status
	}
	if meta.UpstreamResponseTime == 0 {
		meta.UpstreamResponseTime = dur
	}
	return fmt.Sprintf(
		`%s %s -> %s "%s %s %s" %d %s %s`,
		clientIP,
		host,
		target,
		method,
		path,
		proto,
		status,
		formatAccessDuration(dur),
		buildAccessLogExtraFields(req, meta),
	)
}

func formatAccessDuration(d time.Duration) string {
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func accessLogClientIP(req *http.Request) string {
	if req == nil {
		return "-"
	}
	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil && host != "" {
		return host
	}
	if req.RemoteAddr != "" {
		return req.RemoteAddr
	}
	return "-"
}

func handlerAccessTarget(string) string {
	return "handler"
}

func buildAccessLogExtraFields(req *http.Request, meta accessLogMeta) string {
	realIP := req.Header.Get("X-Real-IP")
	if realIP == "" {
		realIP = accessLogClientIP(req)
	}

	referer := req.Referer()
	if referer == "" {
		referer = "-"
	}

	userAgent := req.UserAgent()
	if userAgent == "" {
		userAgent = "-"
	}

	xForwardedFor := req.Header.Get("X-Forwarded-For")
	if xForwardedFor == "" {
		xForwardedFor = "-"
	}

	tlsProtocol := "-"
	tlsCipher := "-"
	if req.TLS != nil {
		tlsProtocol = tls.VersionName(req.TLS.Version)
		tlsCipher = tls.CipherSuiteName(req.TLS.CipherSuite)
		if tlsProtocol == "" {
			tlsProtocol = "-"
		}
		if tlsCipher == "" {
			tlsCipher = "-"
		}
	}

	cacheHit := 0
	if meta.CacheHit {
		cacheHit = 1
	}
	wafBlock := 0
	if meta.WAFBlock {
		wafBlock = 1
	}
	rateLimitBlock := 0
	if meta.RateLimitBlock {
		rateLimitBlock = 1
	}
	maintenanceBlock := 0
	if meta.MaintenanceBlock {
		maintenanceBlock = 1
	}

	return fmt.Sprintf(
		"cache_hit=%d waf_block=%d rate_limit_block=%d maintenance_block=%d real_ip=%s referer=%s ua=%s xff=%s tls_protocol=%s tls_cipher=%s upstream_status=%d upstream_response_length=%d upstream_response_time=%s",
		cacheHit,
		wafBlock,
		rateLimitBlock,
		maintenanceBlock,
		accessLogFieldValue(realIP),
		accessLogFieldValue(referer),
		accessLogFieldValue(userAgent),
		accessLogFieldValue(xForwardedFor),
		accessLogFieldValue(tlsProtocol),
		accessLogFieldValue(tlsCipher),
		meta.UpstreamStatus,
		meta.UpstreamResponseLength,
		formatAccessDuration(meta.UpstreamResponseTime),
	)
}

func accessLogFieldValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	if strings.ContainsAny(v, " \t") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}
