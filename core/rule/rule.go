package rule

import (
	"github.com/go-zoox/ingress/core/service"
)

type Rule struct {
	Host    string  `config:"host"`
	Backend Backend `config:"backend"`
	//
	Paths []Path `config:"paths"`
}

type Backend struct {
	Service service.Service `config:"service"`
}

type Path struct {
	Path    string  `config:"path"`
	Backend Backend `config:"backend"`
}
