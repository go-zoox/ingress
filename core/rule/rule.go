package rule

import (
	"github.com/go-zoox/ingress/core/service"
)

type Rule struct {
	Host    string  `config:"host"`
	Backend Backend `config:"backend"`
	//
	Paths []Path `config:"paths"`
	// HostType is the host match type of Rule, options: exact, regex, wildcard
	HostType string `config:"host_type,default=exact"`
}

type Backend struct {
	Type string `config:"type,default=service"`
	//
	Service service.Service `config:"service"`
	//
	Handler Handler `config:"handler"`
	//
	Redirect Redirect `config:"redirect"`
}

type Path struct {
	Path    string  `config:"path"`
	Backend Backend `config:"backend"`
}

type Redirect struct {
	URL       string `config:"url"`
	Permanent bool   `config:"permanent"`
}

type Handler struct {
	StatusCode int64             `config:"status_code,default=200"`
	Headers    map[string]string `config:"headers"`
	Body       string            `config:"body"`
}
