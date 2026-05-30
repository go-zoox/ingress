package core

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/jobs"
	"github.com/go-zoox/ingress/core/rule"
)

type Config struct {
	Port int64 `config:"port"`
	// EnableH2C enables cleartext HTTP/2 (h2c) on the plaintext HTTP port. Unsafe on public networks; use behind a trusted load balancer or for local testing.
	EnableH2C bool `config:"enable_h2c"`
	//
	Rules []rule.Rule `config:"rules"`
	// WAF is optional global baseline; rules[].waf patches this map-wise (see docs/guide/waf.md).
	WAF rule.WAF `config:"waf"`
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
	// ErrorPages configures built-in and custom HTML for 401/403/404/500/502/503/504.
	ErrorPages ErrorPages `config:"error_pages"`
	//
	// Logger is zoox logger config (YAML key `logging` for historical reasons).
	Logging Logging `config:"logging"`
	//
	Admin Admin `config:"admin"`
	//
	RateLimit rule.RateLimit `config:"rate_limit"`
	// Security optional global baseline for HTTP security response headers (HSTS, CORS, frame, etc.).
	Security rule.Security `config:"security"`
	// Maintenance global host registry and default 503 settings (see maintenance.hosts).
	Maintenance MaintenanceConfig `config:"maintenance"`
	// Jobs configures scheduled tasks (built-in ops jobs and custom platform jobs).
	Jobs jobs.Config `config:"jobs"`
	// Scenarios optional named overlays; scenarios.active selects the effective overlay at runtime.
	Scenarios Scenarios `config:"scenarios"`
	// Match func(host string, path string) (cfg *service.Service, err error)
}

// Admin configures the embedded operations console (HTTP API + optional UI).
type Admin struct {
	Enabled       bool          `config:"enabled"`
	Port          int64         `config:"port,default=9080"`
	Database      AdminDatabase `config:"database"`
	Web           AdminWeb      `config:"web"`
	Auth          AdminAuth     `config:"auth"`
	AccessLogPath string        `config:"access_log_path"`
	ErrorLogPath  string        `config:"error_log_path"`
	GeoIP         AdminGeoIP    `config:"geoip"`
	Jobs          jobs.AdminJobs `config:"jobs"`
}

// AdminAuth configures Admin Console authentication (none | basic | oauth).
type AdminAuth struct {
	// Type is none (default), basic, or oauth.
	Type  string          `config:"type,default=none"`
	Basic AdminAuthBasic  `config:"basic"`
	OAuth AdminAuthOAuth  `config:"oauth"`
}

// AdminAuthBasic configures local username/password login backed by RBAC users.
type AdminAuthBasic struct {
	// Username and Password identify the bootstrap super-admin RBAC user (admin role).
	Username string `config:"username"`
	Password string `config:"password"`
}

// AdminAuthOAuth configures third-party OAuth login for the Admin Console.
type AdminAuthOAuth struct {
	Provider     string   `config:"provider"`
	ClientID     string   `config:"client_id"`
	ClientSecret string   `config:"client_secret"`
	RedirectURL  string   `config:"redirect_url"`
	Scopes       []string `config:"scopes"`
}

// EffectiveAdminAuthType returns none (default), basic, or oauth.
func EffectiveAdminAuthType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "none", "basic", "oauth":
		return strings.ToLower(strings.TrimSpace(t))
	default:
		return "none"
	}
}

// Validate checks admin.auth settings when admin is enabled.
func (a AdminAuth) Validate() error {
	switch EffectiveAdminAuthType(a.Type) {
	case "none", "basic":
		return nil
	case "oauth":
		if strings.TrimSpace(a.OAuth.Provider) == "" {
			return fmt.Errorf("admin.auth.oauth.provider is required when admin.auth.type is oauth")
		}
		if strings.TrimSpace(a.OAuth.ClientID) == "" {
			return fmt.Errorf("admin.auth.oauth.client_id is required when admin.auth.type is oauth")
		}
		if strings.TrimSpace(a.OAuth.ClientSecret) == "" {
			return fmt.Errorf("admin.auth.oauth.client_secret is required when admin.auth.type is oauth")
		}
		return nil
	default:
		return fmt.Errorf("admin.auth.type must be none, basic, or oauth")
	}
}

// AdminGeoIP configures MaxMind GeoLite2 for WAF attack map geolocation.
type AdminGeoIP struct {
	// Database is the path to GeoLite2-City.mmdb (relative to ingress.yaml directory, or absolute).
	Database string `config:"database,default=/etc/geoip/GeoLite2-City.mmdb"`
	// IngressLat/Lng/Label override the defender node on the attack map.
	IngressLat   float64 `config:"ingress_lat"`
	IngressLng   float64 `config:"ingress_lng"`
	IngressLabel string  `config:"ingress_label"`
}

type AdminDatabase struct {
	Driver string `config:"driver,default=sqlite"`
	DSN    string `config:"dsn,default=file:admin.db?cache=shared&_fk=1"`
}

type AdminWeb struct {
	// DevProxy when true serves API only; frontend runs on Vite dev server.
	DevProxy bool `config:"dev_proxy"`
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
	// Enabled activates forced HTTP -> HTTPS redirects when https.port is configured. Default false means no redirect.
	Enabled bool `config:"enabled"`
	// Permanent uses 301 when true; 302 when false (or 308/307 when WithOriginMethodAndBody is true).
	Permanent bool `config:"permanent"`
	// WithOriginMethodAndBody uses HTTP 307/308 so clients preserve method and body on redirect.
	// Default false uses 302/301.
	WithOriginMethodAndBody bool `config:"with_origin_method_and_body"`
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
