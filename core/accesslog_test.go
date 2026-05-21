package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crypto/tls"
)

func TestFormatAccessLog_demoShape(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/api/users", nil)
	req.RemoteAddr = "203.0.113.44:52341"
	req.Header.Set("X-Real-IP", "203.0.113.44")
	req.Header.Set("User-Agent", "curl/8.0")
	req.TLS = &tls.ConnectionState{
		Version:     tls.VersionTLS13,
		CipherSuite: tls.TLS_AES_128_GCM_SHA256,
	}

	line := formatAccessLog(req, "api.example.com", "api.internal:8080", http.MethodGet, "/api/users", "HTTP/1.1", 200, 12*time.Millisecond, accessLogMeta{
		UpstreamStatus:         200,
		UpstreamResponseLength: 456,
		UpstreamResponseTime:   12 * time.Millisecond,
	})

	required := []string{
		`203.0.113.44 api.example.com -> api.internal:8080 "GET /api/users HTTP/1.1" 200 12ms`,
		`cache_hit=0`,
		`waf_block=0`,
		`real_ip=203.0.113.44`,
		`upstream_status=200`,
		`upstream_response_length=456`,
		`upstream_response_time=12ms`,
		`tls_protocol="TLS 1.3"`,
	}
	for _, item := range required {
		if !strings.Contains(line, item) {
			t.Fatalf("expected %q in %q", item, line)
		}
	}
}

func TestBuild_AccessLogExtraFields_WithTLS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/orders?id=1", nil)
	req.Header.Set("Referer", "https://portal.example.com/list")
	req.Header.Set("User-Agent", "ingress-test-agent")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.Header.Set("X-Real-IP", "203.0.113.8")
	req.TLS = &tls.ConnectionState{
		Version:     tls.VersionTLS13,
		CipherSuite: tls.TLS_AES_128_GCM_SHA256,
	}

	extra := buildAccessLogExtraFields(req, accessLogMeta{
		UpstreamStatus:         200,
		UpstreamResponseLength: 456,
		UpstreamResponseTime:   123 * time.Millisecond,
	})

	required := []string{
		`real_ip=203.0.113.8`,
		`referer=https://portal.example.com/list`,
		`ua=ingress-test-agent`,
		`xff="10.0.0.1, 10.0.0.2"`,
		`tls_protocol="TLS 1.3"`,
		`tls_cipher=TLS_AES_128_GCM_SHA256`,
		`upstream_status=200`,
		`upstream_response_length=456`,
		`upstream_response_time=123ms`,
		`cache_hit=0`,
		`waf_block=0`,
	}

	for _, item := range required {
		if !strings.Contains(extra, item) {
			t.Fatalf("expected extra fields to contain %q, got: %s", item, extra)
		}
	}
}

func TestBuild_AccessLogExtraFields_WithoutTLS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)
	req.RemoteAddr = "198.51.100.9:4321"

	extra := buildAccessLogExtraFields(req, accessLogMeta{
		UpstreamStatus:         503,
		UpstreamResponseLength: -1,
		UpstreamResponseTime:   27 * time.Millisecond,
	})

	required := []string{
		`real_ip=198.51.100.9`,
		`referer=-`,
		`ua=-`,
		`xff=-`,
		`tls_protocol=-`,
		`tls_cipher=-`,
		`upstream_status=503`,
		`upstream_response_length=-1`,
		`upstream_response_time=27ms`,
	}

	for _, item := range required {
		if !strings.Contains(extra, item) {
			t.Fatalf("expected extra fields to contain %q, got: %s", item, extra)
		}
	}
}

func TestBuild_AccessLogExtraFields_RealIPFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/ping", nil)
	req.RemoteAddr = "198.51.100.77:9000"
	req.Header.Set("X-Real-IP", "203.0.113.8")

	extra := buildAccessLogExtraFields(req, accessLogMeta{UpstreamStatus: 200, UpstreamResponseTime: 5 * time.Millisecond})
	if !strings.Contains(extra, `real_ip=203.0.113.8`) {
		t.Fatalf("expected real_ip from X-Real-IP, got: %s", extra)
	}
}
