package core

import (
	"errors"
	"fmt"
	"regexp"
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
}

func (c *core) match(ctx *zoox.Context, host string, path string) (s *service.Service, r *rule.Rule, pathBackend *rule.Backend, err error) {
	key := fmt.Sprintf("match.host:%s", host)
	matcher := &HostMatcher{}
	if err := ctx.Cache().Get(key, matcher); err != nil {
		matcher, err = MatchHost(c.cfg.Rules, c.cfg.Fallback, host)
		if err != nil {
			if !errors.Is(err, ErrHostNotFound) {
				return nil, nil, nil, err
			}

			// ctx.Cache().Set(key, nil, 60*time.Second)
			return nil, nil, nil, err
		}

		ctx.Cache().Set(key, matcher, time.Duration(c.cfg.Cache.TTL)*time.Second)
	}

	// host service
	s = matcher.Service
	t := matcher.Rule
	var matchedPathBackend *rule.Backend

	// @TODO not found
	if s == nil {
		return nil, nil, nil, fmt.Errorf("service not found at matcher")
	}

	// paths
	if matcher.IsPathsExist {
		ps, matchedPath, err := MatchPath(matcher.Rule.Paths, path)
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

	// // match func
	// if s == nil {
	// 	if c.cfg.Match != nil {
	// 		sm, err := c.cfg.Match(host, path)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		s = sm
	// 	}
	// }

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

func MatchHost(rules []rule.Rule, fallback rule.Backend, host string) (hm *HostMatcher, err error) {
	for _, rule := range rules {
		switch rule.HostType {
		case "exact", "":
			if rule.Host == host {
				return &HostMatcher{
					Service: &service.Service{
						Protocol: rule.Backend.Service.Protocol,
						Name:     rule.Backend.Service.Name,
						Port:     rule.Backend.Service.Port,
						Request:  rule.Backend.Service.Request,
						Response: rule.Backend.Service.Response,
					},
					IsPathsExist: len(rule.Paths) != 0,
					Rule:         &rule,
				}, nil
			}
		case "regex":
			if isMatched, _ := regexp.MatchString(rule.Host, host); isMatched {
				hostRewriter := rewriter.Rewriter{
					From: rule.Host,
					To:   rule.Backend.Service.Name,
				}

				return &HostMatcher{
					Service: &service.Service{
						Protocol: rule.Backend.Service.Protocol,
						Name:     hostRewriter.Rewrite(host),
						Port:     rule.Backend.Service.Port,
						Request:  rule.Backend.Service.Request,
						Response: rule.Backend.Service.Response,
					},
					IsPathsExist: len(rule.Paths) != 0,
					Rule:         &rule,
				}, nil
			}
		case "wildcard":
			re := wildCardToRegexp(rule.Host)
			// fmt.Println("wildcard", rule.Host, re)
			if isMatched, _ := regexp.MatchString(re, host); isMatched {
				hostRewriter := rewriter.Rewriter{
					From: re,
					To:   rule.Backend.Service.Name,
				}

				return &HostMatcher{
					Service: &service.Service{
						Protocol: rule.Backend.Service.Protocol,
						Name:     hostRewriter.Rewrite(host),
						Port:     rule.Backend.Service.Port,
						Request:  rule.Backend.Service.Request,
						Response: rule.Backend.Service.Response,
					},
					IsPathsExist: len(rule.Paths) != 0,
					Rule:         &rule,
				}, nil
			}
		default:
			return nil, fmt.Errorf("unsupport host type: %s", rule.HostType)
		}
	}

	if fallback.Service.Name != "" {
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
		}, nil
	}

	return nil, ErrHostNotFound
}

func MatchPath(paths []rule.Path, path string) (r *service.Service, matchedPath *rule.Path, err error) {
	for _, rpath := range paths {
		rpathRe := fmt.Sprintf("^%s", rpath.Path)
		//
		isMatched, err := regexp.MatchString(rpathRe, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to match path: %s", err)
		}

		if isMatched {
			return &service.Service{
				Protocol: rpath.Backend.Service.Protocol,
				Name:     rpath.Backend.Service.Name,
				Port:     rpath.Backend.Service.Port,
				Request:  rpath.Backend.Service.Request,
				Response: rpath.Backend.Service.Response,
			}, &rpath, nil
		}
	}

	return nil, nil, ErrPathNotFound
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
