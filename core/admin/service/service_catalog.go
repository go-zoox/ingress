package service

import (
	"strconv"
	"strings"

	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	coresvc "github.com/go-zoox/ingress/core/service"
	"gopkg.in/yaml.v3"
)

// CatalogService is one entry from ingress.yaml services[] (admin catalog).
type CatalogService struct {
	Index       int                 `json:"catalog_index"`
	Name        string              `json:"name"`
	Port        int64               `json:"port"`
	Protocol    string              `json:"protocol"`
	Mode        string              `json:"mode"`
	Note        string              `json:"note"`
	Target      string              `json:"target"`
	HealthCheck coresvc.HealthCheck `json:"-"`
}

// ServiceRouteRef is a rule/path backend that references a service by name.
type ServiceRouteRef struct {
	RuleIndex   int    `json:"rule_index"`
	PathIndex   int    `json:"path_index"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Target      string `json:"target"`
	BackendType string `json:"backend_type"`
}

type catalogYAML struct {
	Services []catalogServiceRow `yaml:"services"`
}

type catalogServiceRow struct {
	Name     string              `yaml:"name"`
	Port     int64               `yaml:"port"`
	Protocol string              `yaml:"protocol"`
	Mode     string              `yaml:"mode"`
	Note     string              `yaml:"note"`
	Health   coresvc.HealthCheck `yaml:"healthcheck"`
}

// ParseServiceCatalog reads services[] from raw ingress YAML (unknown keys ignored).
func ParseServiceCatalog(content string) ([]CatalogService, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}
	var doc catalogYAML
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, err
	}
	out := make([]CatalogService, 0, len(doc.Services))
	for i, row := range doc.Services {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		protocol := strings.TrimSpace(row.Protocol)
		if protocol == "" {
			protocol = "http"
		}
		out = append(out, CatalogService{
			Index:       i,
			Name:        name,
			Port:        row.Port,
			Protocol:    protocol,
			Mode:        strings.TrimSpace(row.Mode),
			Note:        strings.TrimSpace(row.Note),
			Target:      ServiceTarget(name, row.Port, protocol),
			HealthCheck: row.Health,
		})
	}
	return out, nil
}

// FindCatalogService returns the catalog entry for name (exact match).
func FindCatalogService(catalog []CatalogService, name string) (CatalogService, bool) {
	name = strings.TrimSpace(name)
	for _, s := range catalog {
		if s.Name == name {
			return s, true
		}
	}
	return CatalogService{}, false
}

// ServiceTarget builds the access-log upstream target label name:port.
func ServiceTarget(name string, port int64, protocol string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if port <= 0 {
		if strings.EqualFold(strings.TrimSpace(protocol), "https") {
			port = 443
		} else {
			port = 80
		}
	}
	return name + ":" + strconv.FormatInt(port, 10)
}

// ListServiceRouteRefs lists rule/path backends whose service.name matches.
func ListServiceRouteRefs(cfg *ingcore.Config, serviceName string) []ServiceRouteRef {
	if cfg == nil {
		return []ServiceRouteRef{}
	}
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return []ServiceRouteRef{}
	}
	var out []ServiceRouteRef
	for ri := range cfg.Rules {
		r := &cfg.Rules[ri]
		if ref, ok := serviceRouteRefFromBackend(ri, -1, r.Host, "/", r.Backend, serviceName); ok {
			out = append(out, ref)
		}
		for pi := range r.Paths {
			p := &r.Paths[pi]
			path := p.Path
			if path == "" {
				path = "/"
			}
			if ref, ok := serviceRouteRefFromBackend(ri, pi, r.Host, path, p.Backend, serviceName); ok {
				out = append(out, ref)
			}
		}
	}
	if out == nil {
		return []ServiceRouteRef{}
	}
	return out
}

func serviceRouteRefFromBackend(ri, pi int, host, path string, b rule.Backend, serviceName string) (ServiceRouteRef, bool) {
	bt := backendTypeLabel(b)
	if bt != "service" {
		return ServiceRouteRef{}, false
	}
	if strings.TrimSpace(b.Service.Name) != serviceName {
		return ServiceRouteRef{}, false
	}
	target := backendTargetLabel(b)
	return ServiceRouteRef{
		RuleIndex:   ri,
		PathIndex:   pi,
		Host:        host,
		Path:        path,
		Target:      target,
		BackendType: bt,
	}, true
}

// ServiceTargetAliases returns distinct access-log target strings for a service.
func ServiceTargetAliases(catalog CatalogService, refs []ServiceRouteRef) []string {
	seen := map[string]struct{}{}
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		seen[t] = struct{}{}
	}
	add(catalog.Target)
	for _, ref := range refs {
		add(ref.Target)
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	return out
}

func targetSet(aliases []string) map[string]struct{} {
	set := make(map[string]struct{}, len(aliases))
	for _, t := range aliases {
		t = strings.TrimSpace(t)
		if t != "" {
			set[t] = struct{}{}
		}
	}
	return set
}
