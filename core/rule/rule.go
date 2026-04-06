package rule

import (
	"github.com/go-zoox/ingress/core/service"
)

type Rule struct {
	Host    string  `config:"host"`
	Backend Backend `config:"backend"`
	//
	Paths []Path `config:"paths"`
	// HostType is the host match type: exact, regex, wildcard, or auto (empty).
	// Empty or "auto" selects exact vs regex vs wildcard from Host at compile time.
	// Set "exact" explicitly to match Host as a literal string even if it looks like a pattern.
	HostType string `config:"host_type"`
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
	Type       string            `config:"type,default=static_response"`
	Engine     string            `config:"engine,default=javascript"`
	Script     string            `config:"script"`
	StatusCode int64             `config:"status_code,default=200"`
	Headers    map[string]string `config:"headers"`
	Body       string            `config:"body"`
	RootDir    string            `config:"root_dir"`
	IndexFile  string            `config:"index_file,default=index.html"`
}
