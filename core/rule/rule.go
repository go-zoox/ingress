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
	Service service.Service `config:"service"`
}

type Path struct {
	Path    string  `config:"path"`
	Backend Backend `config:"backend"`
}
