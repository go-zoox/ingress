package core

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/proxy/utils/rewriter"
	"github.com/go-zoox/zoox"
)

type HostMatcher struct {
	Service *service.Service
	//
	IsPathsExist bool
	//
	Rule *rule.Rule
	// ruleIndex is the index in cfg.Rules; -1 when Rule is synthetic (e.g. fallback).
	ruleIndex int
	// hostSubmatches stores regex captures, where index 0 is the full match.
	hostSubmatches []string
}

// getBackendType returns backend.type after normalization (prepare / ValidateConfig run inference).
// Empty still means default upstream mode ("service").
func getBackendType(backend rule.Backend) string {
	if backend.Type == "" {
		return backendTypeService
	}

	return backend.Type
}

func backendNeedsNoUpstream(b rule.Backend) bool {
	bt := getBackendType(b)
	return bt == backendTypeHandler || bt == backendTypeRedirect
}

func (c *core) match(ctx *zoox.Context, host string, path string) (*service.Service, *rule.Rule, *rule.Backend, []string, []string, int, int, error) {
	key := "match.host:v2:" + host
	matcher := &HostMatcher{}
	if err := ctx.Cache().Get(key, matcher); err != nil {
		matcher, err = matchHostIndex(c.router, c.cfg.Rules, c.cfg.Fallback, host)
		if err != nil {
			if !errors.Is(err, ErrHostNotFound) {
				return nil, nil, nil, nil, nil, 0, -1, err
			}

			return nil, nil, nil, nil, nil, 0, -1, err
		}

		ctx.Cache().Set(key, matcher, time.Duration(c.cfg.Cache.TTL)*time.Second)
	}

	// host service
	s := matcher.Service
	t := matcher.Rule
	var matchedPathBackend *rule.Backend

	// Paths may replace upstream Service when the path backend proxies elsewhere—or leave it nil for handler/redirect paths.
	var pathSubmatches []string
	pathIdx := -1
	if matcher.IsPathsExist && matcher.ruleIndex >= 0 {
		ps, matchedPath, psm, pi, err := matchPathWithRouter(c.router, c.cfg.Rules, matcher.ruleIndex, path, host, matcher.hostSubmatches)
		if err != nil {
			if !errors.Is(err, ErrPathNotFound) {
				return nil, nil, nil, nil, nil, matcher.ruleIndex, -1, err
			}
		} else {
			pathSubmatches = psm
			pathIdx = pi
			s = ps
			if matchedPath != nil {
				matchedPathBackend = &matchedPath.Backend
			}
		}
	}

	hostNP := backendNeedsNoUpstream(t.Backend)
	pathNP := matchedPathBackend != nil && backendNeedsNoUpstream(*matchedPathBackend)
	if s == nil && !hostNP && !pathNP {
		return nil, nil, nil, nil, nil, matcher.ruleIndex, pathIdx, fmt.Errorf("service not found at matcher")
	}

	if s == nil && (pathNP || hostNP) {
		return nil, t, matchedPathBackend, matcher.hostSubmatches, pathSubmatches, matcher.ruleIndex, pathIdx, nil
	}

	if s == nil {
		s = &c.cfg.Fallback.Service
		t = &rule.Rule{
			Host:     fallbackRuleHost,
			HostType: hostTypeExact,
			Backend:  c.cfg.Fallback,
		}
		return s, t, matchedPathBackend, matcher.hostSubmatches, pathSubmatches, matcher.ruleIndex, pathIdx, nil
	}

	return s, t, matchedPathBackend, matcher.hostSubmatches, pathSubmatches, matcher.ruleIndex, pathIdx, nil
}

// MatchHost matches host against rules using a precompiled index when available.
// For one-off calls it compiles the index each time; the hot path uses core.router from prepare().
func MatchHost(rules []rule.Rule, fallback rule.Backend, host string) (hm *HostMatcher, err error) {
	if err := inferRuleBackends(rules); err != nil {
		return nil, err
	}
	idx, err := compileRouterIndex(rules, fallback)
	if err != nil {
		return nil, err
	}
	return matchHostIndex(idx, rules, fallback, host)
}

func matchHostIndex(router *routerIndex, rules []rule.Rule, fallback rule.Backend, host string) (*HostMatcher, error) {
	for _, e := range router.entries {
		r := &rules[e.ruleIndex]
		switch e.hostType {
		case hostTypeExact:
			if e.exactHost != host {
				continue
			}
			return hostMatcherFromMatchedRule(r, host, "", nil, e.ruleIndex)
		case hostTypeRegex:
			submatches := e.re.FindStringSubmatch(host)
			if submatches == nil {
				continue
			}
			return hostMatcherFromMatchedRule(r, host, r.Host, submatches, e.ruleIndex)
		case hostTypeWildcard:
			if !e.re.MatchString(host) {
				continue
			}
			return hostMatcherFromMatchedRule(r, host, e.wildcardRewriterFrom, nil, e.ruleIndex)
		}
	}

	if router.fallbackValid {
		return &HostMatcher{
			Service: &service.Service{
				Protocol:    fallback.Service.Protocol,
				Name:        fallback.Service.Name,
				Port:        fallback.Service.Port,
				Mode:        fallback.Service.Mode,
				StripPrefix: fallback.Service.StripPrefix,
				Request:     fallback.Service.Request,
				Response:    fallback.Service.Response,
				Auth:        fallback.Service.Auth,
				HealthCheck: fallback.Service.HealthCheck,
			},
			IsPathsExist: false,
			Rule: &rule.Rule{
				Host:     fallbackRuleHost,
				HostType: hostTypeExact,
				Backend:  fallback,
			},
			ruleIndex: -1,
		}, nil
	}

	return nil, ErrHostNotFound
}

func hostMatcherFromMatchedRule(rule *rule.Rule, host string, rewriterFrom string, hostSubmatches []string, ruleIndex int) (*HostMatcher, error) {
	backendType := getBackendType(rule.Backend)
	switch backendType {
	case backendTypeHandler, backendTypeRedirect:
		return &HostMatcher{
			Service:        nil,
			IsPathsExist:   len(rule.Paths) != 0,
			Rule:           rule,
			ruleIndex:      ruleIndex,
			hostSubmatches: hostSubmatches,
		}, nil
	case backendTypeService:
		if rewriterFrom == "" {
			return &HostMatcher{
				Service: &service.Service{
					Protocol: rule.Backend.Service.Protocol,
					Name: renderServiceName(
						rule.Backend.Service.Name,
						"",
						host,
						hostSubmatches,
						nil,
					),
					Port:        rule.Backend.Service.Port,
					Mode:        rule.Backend.Service.Mode,
					StripPrefix: rule.Backend.Service.StripPrefix,
					Request:     rule.Backend.Service.Request,
					Response:    rule.Backend.Service.Response,
					Auth:        rule.Backend.Service.Auth,
					HealthCheck: rule.Backend.Service.HealthCheck,
				},
				IsPathsExist:   len(rule.Paths) != 0,
				Rule:           rule,
				ruleIndex:      ruleIndex,
				hostSubmatches: hostSubmatches,
			}, nil
		}

		return &HostMatcher{
			Service: &service.Service{
				Protocol: rule.Backend.Service.Protocol,
				Name: renderServiceName(
					rule.Backend.Service.Name,
					rewriterFrom,
					host,
					hostSubmatches,
					nil,
				),
				Port:        rule.Backend.Service.Port,
				Mode:        rule.Backend.Service.Mode,
				StripPrefix: rule.Backend.Service.StripPrefix,
				Request:     rule.Backend.Service.Request,
				Response:    rule.Backend.Service.Response,
				Auth:        rule.Backend.Service.Auth,
				HealthCheck: rule.Backend.Service.HealthCheck,
			},
			IsPathsExist:   len(rule.Paths) != 0,
			Rule:           rule,
			ruleIndex:      ruleIndex,
			hostSubmatches: hostSubmatches,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}

func matchPathWithRouter(router *routerIndex, rules []rule.Rule, ruleIdx int, path string, host string, hostSubmatches []string) (r *service.Service, matchedPath *rule.Path, pathSubmatches []string, pathIndex int, err error) {
	if ruleIdx < 0 || ruleIdx >= len(rules) {
		return nil, nil, nil, -1, ErrPathNotFound
	}
	matchedRule := &rules[ruleIdx]
	for _, cp := range router.pathsByRule[ruleIdx] {
		psubs := cp.re.FindStringSubmatch(path)
		if psubs == nil {
			continue
		}
		rp := &rules[ruleIdx].Paths[cp.pathIndex]
		svc, mp, err := pathMatchResultWithHost(rp, matchedRule, host, hostSubmatches, psubs)
		return svc, mp, psubs, cp.pathIndex, err
	}
	return nil, nil, nil, -1, ErrPathNotFound
}

func pathMatchResult(rpath *rule.Path) (*service.Service, *rule.Path, error) {
	return pathMatchResultWithHost(rpath, nil, "", nil, nil)
}

func pathMatchResultWithHost(rpath *rule.Path, matchedRule *rule.Rule, host string, hostSubmatches []string, pathSubmatches []string) (*service.Service, *rule.Path, error) {
	backendType := getBackendType(rpath.Backend)
	if backendType == backendTypeHandler || backendType == backendTypeRedirect {
		return nil, rpath, nil
	}
	if backendType != backendTypeService {
		return nil, nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}

	name := rpath.Backend.Service.Name
	rewriterFrom := ""
	if matchedRule != nil && matchedRule.HostType == hostTypeRegex {
		rewriterFrom = matchedRule.Host
		if len(hostSubmatches) == 0 && host != "" {
			re, err := regexp.Compile(matchedRule.Host)
			if err == nil {
				hostSubmatches = re.FindStringSubmatch(host)
			}
		}
	}
	name = renderServiceName(name, rewriterFrom, host, hostSubmatches, pathSubmatches)

	return &service.Service{
		Protocol:    rpath.Backend.Service.Protocol,
		Name:        name,
		Port:        rpath.Backend.Service.Port,
		Mode:        rpath.Backend.Service.Mode,
		StripPrefix: rpath.Backend.Service.StripPrefix,
		Request:     rpath.Backend.Service.Request,
		Response:    rpath.Backend.Service.Response,
		Auth:        rpath.Backend.Service.Auth,
		HealthCheck: rpath.Backend.Service.HealthCheck,
	}, rpath, nil
}

func MatchPath(paths []rule.Path, path string) (r *service.Service, matchedPath *rule.Path, err error) {
	if err := inferPathSliceBackends(paths, 0, ""); err != nil {
		return nil, nil, err
	}
	for i := range paths {
		rpath := &paths[i]
		re, err := regexp.Compile("^" + rpath.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to match path: %s", err)
		}
		if re.MatchString(path) {
			return pathMatchResult(rpath)
		}
	}

	return nil, nil, ErrPathNotFound
}

var serviceNameTemplateRegexp = regexp.MustCompile(`\$\{(host|path)\.(\d+)\}`)

func renderServiceName(raw string, legacyHostPattern string, host string, hostSubmatches []string, pathSubmatches []string) string {
	name := serviceNameTemplateRegexp.ReplaceAllStringFunc(raw, func(token string) string {
		match := serviceNameTemplateRegexp.FindStringSubmatch(token)
		if len(match) != 3 {
			return token
		}
		index, err := strconv.Atoi(match[2])
		if err != nil {
			return token
		}

		var captures []string
		switch match[1] {
		case "host":
			captures = hostSubmatches
		case "path":
			captures = pathSubmatches
		default:
			return token
		}

		if index < 0 || index >= len(captures) {
			return token
		}
		return captures[index]
	})

	// Backward compatibility for legacy $1/$2 host capture syntax.
	if legacyHostPattern != "" && host != "" {
		hostRewriter := rewriter.Rewriter{
			From: legacyHostPattern,
			To:   name,
		}
		name = hostRewriter.Rewrite(host)
	}

	return name
}

// expandRedirectURL applies the same capture templating as backend service names:
// ${host.N}, ${path.N}, and legacy $1/$2 via regexp.ReplaceAllString when host_type is regex or wildcard.
func expandRedirectURL(rule *rule.Rule, host string, raw string, hostSubmatches, pathSubmatches []string) string {
	if raw == "" || rule == nil {
		return raw
	}
	legacy := redirectLegacyHostPattern(rule)
	return renderServiceName(raw, legacy, host, hostSubmatches, pathSubmatches)
}

func redirectLegacyHostPattern(rule *rule.Rule) string {
	switch rule.HostType {
	case hostTypeRegex:
		return rule.Host
	case hostTypeWildcard:
		return wildCardToRegexp(rule.Host)
	default:
		return ""
	}
}

// stackoverflow: https://stackoverflow.com/questions/64509506/golang-determine-if-string-contains-a-string-with-wildcards
func wildCardToRegexp(pattern string) string {
	components := strings.Split(pattern, "*")
	if len(components) == 1 {
		// if len is 1, there are no *'s, return exact match pattern
		return "^" + pattern + "$"
	}
	var result strings.Builder
	for i, literal := range components {

		// Replace * with .*
		if i > 0 {
			result.WriteString(".*")
		}

		// Quote any regular expression meta characters in the
		// literal text.
		result.WriteString(regexp.QuoteMeta(literal))
	}
	return "^" + result.String() + "$"
}
