package core

import (
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
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
	Fallback service.Service `config:"fallback"`
	//
	Match func(host string, path string) (cfg *service.Service, err error)
}

type Cache struct {
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
