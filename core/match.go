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

func getBackendType(backend rule.Backend) string {
	if backend.Type == "" {
		return backendTypeService
	}

	return backend.Type
}

func (c *core) match(ctx *zoox.Context, host string, path string) (s *service.Service, r *rule.Rule, pathBackend *rule.Backend, err error) {
	key := "match.host:v2:" + host
	matcher := &HostMatcher{}
	if err := ctx.Cache().Get(key, matcher); err != nil {
		matcher, err = matchHostIndex(c.router, c.cfg.Rules, c.cfg.Fallback, host)
		if err != nil {
			if !errors.Is(err, ErrHostNotFound) {
				return nil, nil, nil, err
			}

			return nil, nil, nil, err
		}

		ctx.Cache().Set(key, matcher, time.Duration(c.cfg.Cache.TTL)*time.Second)
	}

	// host service
	s = matcher.Service
	t := matcher.Rule
	var matchedPathBackend *rule.Backend

	// service can be nil for backend.type=handler
	if s == nil && getBackendType(matcher.Rule.Backend) != backendTypeHandler {
		return nil, nil, nil, fmt.Errorf("service not found at matcher")
	}

	// paths
	if matcher.IsPathsExist && matcher.ruleIndex >= 0 {
		ps, matchedPath, err := matchPathWithRouter(c.router, c.cfg.Rules, matcher.ruleIndex, path, host)
		if err != nil {
			if !errors.Is(err, ErrPathNotFound) {
				return nil, nil, nil, err
			}
		} else {
			s = ps
			if matchedPath != nil {
				matchedPathBackend = &matchedPath.Backend
			}
		}
	}

	isPathHandlerBackend := matchedPathBackend != nil && getBackendType(*matchedPathBackend) == backendTypeHandler
	isHostHandlerBackend := getBackendType(t.Backend) == backendTypeHandler
	if s == nil && (isPathHandlerBackend || isHostHandlerBackend) {
		return nil, t, matchedPathBackend, nil
	}

	if s == nil {
		s = &c.cfg.Fallback.Service
		// force rewrite host
		s.Request.Host.Rewrite = true
		// @TODO
		t = &rule.Rule{}
		t.HostType = "exact"
	}

	return s, t, matchedPathBackend, nil
}

// MatchHost matches host against rules using a precompiled index when available.
// For one-off calls it compiles the index each time; the hot path uses core.router from prepare().
func MatchHost(rules []rule.Rule, fallback rule.Backend, host string) (hm *HostMatcher, err error) {
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
		case "exact":
			if e.exactHost != host {
				continue
			}
			return hostMatcherFromMatchedRule(r, host, "", nil, e.ruleIndex)
		case "regex":
			submatches := e.re.FindStringSubmatch(host)
			if submatches == nil {
				continue
			}
			return hostMatcherFromMatchedRule(r, host, r.Host, submatches, e.ruleIndex)
		case "wildcard":
			if !e.re.MatchString(host) {
				continue
			}
			return hostMatcherFromMatchedRule(r, host, e.wildcardRewriterFrom, nil, e.ruleIndex)
		}
	}

	if router.fallbackValid {
		return &HostMatcher{
			Service: &service.Service{
				Protocol: fallback.Service.Protocol,
				Name:     fallback.Service.Name,
				Port:     fallback.Service.Port,
				Request:  fallback.Service.Request,
				Response: fallback.Service.Response,
			},
			IsPathsExist: false,
			Rule: &rule.Rule{
				Host:     "@@fallback",
				HostType: "exact",
			},
			ruleIndex: -1,
		}, nil
	}

	return nil, ErrHostNotFound
}

func hostMatcherFromMatchedRule(rule *rule.Rule, host string, rewriterFrom string, hostSubmatches []string, ruleIndex int) (*HostMatcher, error) {
	backendType := getBackendType(rule.Backend)
	if backendType == backendTypeHandler {
		return &HostMatcher{
			Service:      nil,
			IsPathsExist: len(rule.Paths) != 0,
			Rule:         rule,
			ruleIndex:    ruleIndex,
			hostSubmatches: hostSubmatches,
		}, nil
	}
	if backendType != backendTypeService {
		return nil, fmt.Errorf("unsupport backend type: %s", backendType)
	}

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
				Port:     rule.Backend.Service.Port,
				Request:  rule.Backend.Service.Request,
				Response: rule.Backend.Service.Response,
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
			Port:     rule.Backend.Service.Port,
			Request:  rule.Backend.Service.Request,
			Response: rule.Backend.Service.Response,
		},
		IsPathsExist:   len(rule.Paths) != 0,
		Rule:           rule,
		ruleIndex:      ruleIndex,
		hostSubmatches: hostSubmatches,
	}, nil
}

func matchPathWithRouter(router *routerIndex, rules []rule.Rule, ruleIdx int, path string, host string) (r *service.Service, matchedPath *rule.Path, err error) {
	if ruleIdx < 0 || ruleIdx >= len(rules) {
		return nil, nil, ErrPathNotFound
	}
	matchedRule := &rules[ruleIdx]
	for _, cp := range router.pathsByRule[ruleIdx] {
		pathSubmatches := cp.re.FindStringSubmatch(path)
		if pathSubmatches == nil {
			continue
		}
		rp := &rules[ruleIdx].Paths[cp.pathIndex]
		return pathMatchResultWithHost(rp, matchedRule, host, nil, pathSubmatches)
	}
	return nil, nil, ErrPathNotFound
}

func pathMatchResult(rpath *rule.Path) (*service.Service, *rule.Path, error) {
	return pathMatchResultWithHost(rpath, nil, "", nil, nil)
}

func pathMatchResultWithHost(rpath *rule.Path, matchedRule *rule.Rule, host string, hostSubmatches []string, pathSubmatches []string) (*service.Service, *rule.Path, error) {
	backendType := getBackendType(rpath.Backend)
	if backendType == backendTypeHandler {
		return nil, rpath, nil
	}
	if backendType != backendTypeService {
		return nil, nil, fmt.Errorf("unsupport backend type: %s", backendType)
	}

	name := rpath.Backend.Service.Name
	rewriterFrom := ""
	if matchedRule != nil && matchedRule.HostType == "regex" {
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
		Protocol: rpath.Backend.Service.Protocol,
		Name:     name,
		Port:     rpath.Backend.Service.Port,
		Request:  rpath.Backend.Service.Request,
		Response: rpath.Backend.Service.Response,
	}, rpath, nil
}

func MatchPath(paths []rule.Path, path string) (r *service.Service, matchedPath *rule.Path, err error) {
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
