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
	Auth        string `json:"auth"`
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
	Auth        string `json:"auth"`
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
		Auth:        authLabelFromBackend(b),
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

func authLabelFromBackend(b rule.Backend) string {
	if b.Service.Auth.Type == "" {
		return ""
	}
	label := authLabelFromAuthFields(b.Service.Auth)
	if b.Service.Auth.Enabled != nil && !*b.Service.Auth.Enabled {
		label += " (disabled)"
	}
	return label
}

func authLabelFromService(s *service.Service) string {
	if s == nil || s.Auth.Type == "" {
		return ""
	}
	label := authLabelFromAuthFields(s.Auth)
	if s.Auth.Enabled != nil && !*s.Auth.Enabled {
		label += " (disabled)"
	}
	return label
}

func authLabelFromAuthFields(auth service.Auth) string {
	switch auth.Type {
	case "basic":
		cnt := len(auth.Basic.Users)
		return fmt.Sprintf("basic (%d users)", cnt)
	case "bearer":
		return "bearer"
	case "oauth2":
		provider := auth.OAuth2.Provider
		if provider != "" {
			return fmt.Sprintf("oauth2 (%s)", provider)
		}
		return "oauth2"
	default:
		return auth.Type
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
			Auth:        authLabelFromBackend(cfg.Fallback),
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
		Auth:        authLabelFromService(svc),
	}, nil
}

// CacheGlobalView describes top-level cache engine settings.
type CacheGlobalView struct {
	Enabled bool   `json:"enabled"`
	Engine  string `json:"engine"`
	TTL     int64  `json:"ttl"`
	Host    string `json:"host"`
	Port    int64  `json:"port"`
	Prefix  string `json:"prefix"`
}

// CacheRouteRow is a route/path with HTTP response cache enabled.
type CacheRouteRow struct {
	ID          int    `json:"id"`
	RuleIndex   int    `json:"rule_index"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	BackendType string `json:"backend_type"`
	Target      string `json:"target"`
	TTL         int64  `json:"ttl"`
	MaxBodyKB   int64  `json:"max_body_kb"`
	KeyHash     string `json:"key_hash"`
}

// CacheGlobalViewFromConfig builds the global cache panel from ingress config.
func CacheGlobalViewFromConfig(cfg *Config) CacheGlobalView {
	if cfg == nil {
		return CacheGlobalView{Engine: "memory"}
	}
	engine := "memory"
	enabled := cfg.Cache.Host != ""
	if enabled {
		engine = "redis"
	}
	ttl := cfg.Cache.TTL
	if ttl == 0 {
		ttl = 60
	}
	prefix := cfg.Cache.Prefix
	if prefix == "" && enabled {
		prefix = "gozoox-ingress:"
	}
	return CacheGlobalView{
		Enabled: enabled || cfg.Cache.Prefix != "",
		Engine:  engine,
		TTL:     ttl,
		Host:    cfg.Cache.Host,
		Port:    cfg.Cache.Port,
		Prefix:  prefix,
	}
}

// ListCacheRouteRows returns backends with backend.cache.enabled.
func ListCacheRouteRows(cfg *Config) ([]CacheRouteRow, error) {
	if err := inferBackendTypes(cfg); err != nil {
		return nil, err
	}
	var rows []CacheRouteRow
	id := 1
	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		if r.Backend.Cache.Enabled {
			rows = append(rows, cacheRowFromBackend(id, i, "/", r.Host, r.Backend))
			id++
		}
		for j := range r.Paths {
			p := &r.Paths[j]
			if p.Backend.Cache.Enabled {
				rows = append(rows, cacheRowFromBackend(id, i, p.Path, r.Host, p.Backend))
				id++
			}
		}
	}
	return rows, nil
}

func cacheRowFromBackend(id, ruleIndex int, path, host string, b rule.Backend) CacheRouteRow {
	c := b.Cache
	ttl := c.TTL
	if ttl == 0 {
		ttl = 300
	}
	maxKB := c.MaxBodyBytes / 1024
	if maxKB == 0 {
		maxKB = 2048
	}
	kh := c.KeyHash
	if kh == "" {
		kh = "md5"
	}
	return CacheRouteRow{
		ID:          id,
		RuleIndex:   ruleIndex,
		Host:        host,
		Path:        path,
		BackendType: getBackendType(b),
		Target:      backendTargetSummary(b),
		TTL:         ttl,
		MaxBodyKB:   maxKB,
		KeyHash:     kh,
	}
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
