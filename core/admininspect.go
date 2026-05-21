package core

import (
	"fmt"
	"strconv"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/ingress/core/waf"
)

// RouteRow is a flattened routing entry for admin UIs.
type RouteRow struct {
	ID          int    `json:"id"`
	RuleIndex   int    `json:"rule_index"`
	PathIndex   int    `json:"path_index"`
	Host        string `json:"host"`
	HostType    string `json:"host_type"`
	Path        string `json:"path"`
	PathPattern string `json:"path_pattern"`
	BackendType string `json:"backend_type"`
	Target      string `json:"target"`
	WAF         string `json:"waf"`
	Cache       bool   `json:"cache"`
}

// MatchPreview is the result of a dry-run host/path match.
type MatchPreview struct {
	Matched     bool   `json:"matched"`
	RuleIndex   int    `json:"rule_index"`
	Host        string `json:"host"`
	HostType    string `json:"host_type"`
	Path        string `json:"path"`
	BackendType string `json:"backend_type"`
	Target      string `json:"target"`
	WAF         string `json:"waf"`
	Fallback    bool   `json:"fallback"`
	Message     string `json:"message,omitempty"`
}

// PrepareForInspect runs inference and compilation checks used by admin tools.
func PrepareForInspect(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := inferBackendTypes(cfg); err != nil {
		return err
	}
	if _, _, err := waf.CompileIngress(cfg.WAF, cfg.Rules); err != nil {
		return err
	}
	_, err := compileRouterIndex(cfg.Rules, cfg.Fallback)
	return err
}

// ListRouteRows returns display rows for all host-level and path-level backends.
func ListRouteRows(cfg *Config) ([]RouteRow, error) {
	if err := inferBackendTypes(cfg); err != nil {
		return nil, err
	}
	var rows []RouteRow
	id := 1
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		ht := effectiveHostType(r.HostType, r.Host)
		r.HostType = ht
		rows = append(rows, routeRowFromBackend(id, i, -1, r, ht, "/", r.Backend))
		id++
		for j := range r.Paths {
			p := &r.Paths[j]
			rows = append(rows, routeRowFromBackend(id, i, j, r, ht, p.Path, p.Backend))
			id++
		}
	}
	return rows, nil
}

func routeRowFromBackend(id, ruleIndex, pathIndex int, r *rule.Rule, hostType, path string, b rule.Backend) RouteRow {
	bt := getBackendType(b)
	target := backendTargetSummary(b)
	cache := b.Cache.Enabled
	wafLabel := "inherit"
	if len(r.WAFPatch) > 0 {
		wafLabel = "patched"
	}
	return RouteRow{
		ID:          id,
		RuleIndex:   ruleIndex,
		PathIndex:   pathIndex,
		Host:        r.Host,
		HostType:    hostType,
		Path:        path,
		PathPattern: path,
		BackendType: bt,
		Target:      target,
		WAF:         wafLabel,
		Cache:       cache,
	}
}

func backendTargetSummary(b rule.Backend) string {
	switch getBackendType(b) {
	case backendTypeRedirect:
		if b.Redirect.URL != "" {
			return b.Redirect.URL
		}
		return "(redirect)"
	case backendTypeHandler:
		if b.Handler.Type != "" {
			return b.Handler.Type
		}
		return "(handler)"
	default:
		s := b.Service
		if s.Name == "" {
			return ""
		}
		port := s.Port
		if port == 0 {
			if s.Protocol == "https" {
				port = 443
			} else {
				port = 80
			}
		}
		return s.Name + ":" + strconv.FormatInt(port, 10)
	}
}

// PreviewMatch dry-runs routing for host and path without starting a server.
func PreviewMatch(cfg *Config, host, path string) (*MatchPreview, error) {
	if err := inferBackendTypes(cfg); err != nil {
		return nil, err
	}
	idx, err := compileRouterIndex(cfg.Rules, cfg.Fallback)
	if err != nil {
		return nil, err
	}

	hm, err := matchHostIndex(idx, cfg.Rules, cfg.Fallback, host)
	if err != nil {
		if err == ErrHostNotFound {
			return &MatchPreview{Matched: false, Message: "no host rule matched"}, nil
		}
		return nil, err
	}

	t := hm.Rule
	bt := getBackendType(t.Backend)
	target := backendTargetSummary(t.Backend)
	matchedPath := "/"
	var svc *service.Service = hm.Service

	if hm.IsPathsExist && hm.ruleIndex >= 0 {
		ps, mp, _, perr := matchPathWithRouter(idx, cfg.Rules, hm.ruleIndex, path, host, hm.hostSubmatches)
		if perr != nil {
			if perr == ErrPathNotFound {
				return &MatchPreview{
					Matched:   false,
					RuleIndex: hm.ruleIndex,
					Host:      t.Host,
					HostType:  t.HostType,
					Message:   "host matched but path did not",
				}, nil
			}
			return nil, perr
		}
		svc = ps
		if mp != nil {
			matchedPath = mp.Path
			bt = getBackendType(mp.Backend)
			target = backendTargetSummary(mp.Backend)
		}
	}

	if hm.ruleIndex < 0 {
		return &MatchPreview{
			Matched:     true,
			RuleIndex:   -1,
			Host:        fallbackRuleHost,
			HostType:    hostTypeExact,
			Path:        matchedPath,
			BackendType: backendTypeService,
			Target:      backendTargetSummary(cfg.Fallback),
			Fallback:    true,
		}, nil
	}

	if svc != nil {
		target = serviceTarget(svc)
	}

	return &MatchPreview{
		Matched:     true,
		RuleIndex:   hm.ruleIndex,
		Host:        t.Host,
		HostType:    t.HostType,
		Path:        matchedPath,
		BackendType: bt,
		Target:      target,
		WAF:         "inherit",
	}, nil
}

func serviceTarget(s *service.Service) string {
	if s == nil {
		return ""
	}
	port := s.Port
	if port == 0 {
		if s.Protocol == "https" {
			port = 443
		} else {
			port = 80
		}
	}
	return s.Name + ":" + strconv.FormatInt(port, 10)
}
