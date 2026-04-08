package core

import (
	"github.com/go-zoox/ingress/core/rule"
)

type Config struct {
	Port int64 `config:"port"`
	// EnableH2C enables cleartext HTTP/2 (h2c) on the plaintext HTTP port. Unsafe on public networks; use behind a trusted load balancer or for local testing.
	EnableH2C bool `config:"enable_h2c"`
	//
	Rules []rule.Rule `config:"rules"`
	//
	Cache Cache `config:"cache"`
	//
	HTTPS HTTPS `config:"https"`
	//
	Fallback rule.Backend `config:"fallback"`
	//
	HealthCheck HealthCheck `config:"healthcheck"`
	//
	// ErrorPageExposeDetails includes host, path, method, and error strings in route-miss
	// HTML responses. Leave false (default) on public-facing ingress; enable only for
	// staging or trusted networks to avoid leaking internal details to clients.
	ErrorPageExposeDetails bool `config:"error_page_expose_details"`
	//
	// Match func(host string, path string) (cfg *service.Service, err error)
}

type HTTPS struct {
	Port int64 `config:"port"`
	SSL  []SSL `config:"ssl"`
	// RedirectFromHTTP enforces global HTTP -> HTTPS redirects before route handling.
	RedirectFromHTTP RedirectFromHTTP `config:"redirect_from_http"`
	// EnableHTTP3 starts an HTTP/3 (QUIC) listener on UDP when https.port is set and TLS is available. Clients discover it via Alt-Svc on HTTPS responses unless http3_altsvc_max_age is negative.
	EnableHTTP3 bool `config:"enable_http3"`
	// HTTP3Port is the UDP port for HTTP/3. Zero means the same port as https.port.
	HTTP3Port int64 `config:"http3_port"`
	// HTTP3AltSvcMaxAge is the ma= value (seconds) for the Alt-Svc header. Zero lets the framework default apply; negative disables Alt-Svc.
	HTTP3AltSvcMaxAge int64 `config:"http3_altsvc_max_age"`
}

type RedirectFromHTTP struct {
	// Disabled controls forced HTTP -> HTTPS redirects. Default false means enabled.
	Disabled bool `config:"disabled"`
	// Permanent uses 301 when true; 302 when false.
	Permanent bool `config:"permanent"`
	// ExcludePaths skips redirect for exact path matches.
	ExcludePaths []string `config:"exclude_paths"`
}

type HealthCheck struct {
	Outer HealthCheckOuter `config:"outer"`
	Inner HealthCheckInner `config:"inner"`
}

type HealthCheckOuter struct {
	Enable bool `config:"enable"`
	// Path is the health check request path
	Path string `config:"path"`
	// Ok means all health check request returns ok
	Ok bool `config:"ok"`
}

type HealthCheckInner struct {
	Enable bool `config:"enable"`
	//
	Interval int64 `config:"interval"`
	Timeout  int64 `config:"timeout"`
}

type Cache struct {
	// TTL is the cache ttl in seconds, default is 60 seconds
	TTL int64 `config:"ttl"`
	//
	Host     string `config:"host"`
	Port     int64  `config:"port"`
	Username string `config:"username"`
	Password string `config:"password"`
	DB       int64  `config:"db"`
	Prefix   string `config:"prefix"`
}

type SSL struct {
	Domain string  `config:"domain"`
	Cert   SSLCert `config:"cert"`
}

type SSLCert struct {
	Certificate    string `config:"certificate"`
	CertificateKey string `config:"certificate_key"`
}
