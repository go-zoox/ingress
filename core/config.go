package core

import (
	"github.com/go-zoox/ingress/core/rule"
)

type Config struct {
	Port int64 `config:"port"`
	//
	Rules []rule.Rule `config:"rules"`
	//
	Cache Cache `config:"cache"`
	//
	HTTPSPort int64 `config:"https_port"`
	SSL       []SSL `config:"ssl"`
	//
	Fallback rule.Backend `config:"fallback"`
	//
	HealthCheck HealthCheck `config:"healthcheck"`
	//
	// Match func(host string, path string) (cfg *service.Service, err error)
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
