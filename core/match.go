package core

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/proxy/utils/rewriter"
	"github.com/go-zoox/zoox"
)

type HostMatcher struct {
	Service service.Service
	//
	IsPathsExist bool
	//
	Rule rule.Rule
}

func (c *core) match(ctx *zoox.Context, host string, path string) (s *service.Service, err error) {
	key := fmt.Sprintf("match.host:%s", host)
	matcher := &HostMatcher{}
	if err := ctx.Cache().Get(key, matcher); err != nil {
		matcher, err = MatchHost(c.cfg.Rules, host)
		if err != nil {
			if !errors.Is(err, ErrHostNotFound) {
				return nil, err
			}

			// ctx.Cache().Set(key, nil, 60*time.Second)
			return nil, nil
		}

		ctx.Cache().Set(key, matcher, 60*time.Second)
	}

	// host service
	s = &matcher.Service

	// paths
	if matcher.IsPathsExist {
		ps, err := MatchPath(matcher.Rule.Paths, path)
		if err != nil {
			if !errors.Is(err, ErrPathNotFound) {
				return nil, err
			}
		} else {
			s = ps
		}
	}

	// match func
	if s == nil {
		if c.cfg.Match != nil {
			sm, err := c.cfg.Match(host, path)
			if err != nil {
				return nil, err
			}

			s = sm
		}
	}

	if s == nil {
		s = &c.cfg.Fallback
		// force rewrite host
		s.Request.Host.Rewrite = true
	}

	return s, nil
}

func MatchHost(rules []rule.Rule, host string) (hm *HostMatcher, err error) {
	for _, rule := range rules {
		hostRegExp := fmt.Sprintf("^%s$", rule.Host)
		if isMatched, _ := regexp.MatchString(hostRegExp, host); isMatched {
			hostRewriter := rewriter.Rewriter{
				From: hostRegExp,
				To:   rule.Backend.Service.Name,
			}

			return &HostMatcher{
				Service: service.Service{
					Protocol: rule.Backend.Service.Protocol,
					Name:     hostRewriter.Rewrite(host),
					Port:     rule.Backend.Service.Port,
					Request:  rule.Backend.Service.Request,
					Response: rule.Backend.Service.Response,
				},
				IsPathsExist: len(rule.Paths) != 0,
				Rule:         rule,
			}, nil
		}
	}

	return nil, ErrHostNotFound
}

func MatchPath(paths []rule.Path, path string) (r *service.Service, err error) {
	for _, rpath := range paths {
		rpathRe := fmt.Sprintf("^%s", rpath.Path)
		//
		isMatched, err := regexp.MatchString(rpathRe, path)
		if err != nil {
			return nil, fmt.Errorf("failed to match path: %s", err)
		}

		if isMatched {
			return &service.Service{
				Protocol: rpath.Backend.Service.Protocol,
				Name:     rpath.Backend.Service.Name,
				Port:     rpath.Backend.Service.Port,
				Request:  rpath.Backend.Service.Request,
				Response: rpath.Backend.Service.Response,
			}, nil
		}
	}

	return nil, ErrPathNotFound
}
