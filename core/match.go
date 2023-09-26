package core

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/proxy/utils/rewriter"
)

func (c *core) match(host string, path string) (s *service.Service, err error) {
	s, err = Match(c.cfg.Rules, host, path)
	if err != nil {
		return nil, err
	}

	if s == nil {
		if c.cfg.Match != nil {
			return c.cfg.Match(host, path)
		}
	}

	return s, nil
}

func Match(rules []rule.Rule, host, path string) (s *service.Service, err error) {
	hostService, rule, err := matchHost(rules, host)
	if err != nil {
		if !errors.Is(err, ErrHostNotFound) {
			return nil, err
		}

		return nil, nil
	}

	pathService, err := matchPath(rule.Paths, path)
	if err != nil {
		if !errors.Is(err, ErrPathNotFound) {
			return nil, err
		}

		return hostService, nil
	}

	return pathService, nil
}

func matchHost(rules []rule.Rule, host string) (b *service.Service, r *rule.Rule, err error) {
	for _, rule := range rules {
		hostRegExp := fmt.Sprintf("^%s$", rule.Host)
		if isMatched, _ := regexp.MatchString(hostRegExp, host); isMatched {
			hostRewriter := rewriter.Rewriter{
				From: hostRegExp,
				To:   rule.Backend.Service.Name,
			}
			s := &service.Service{
				Protocol: rule.Backend.Service.Protocol,
				Name:     hostRewriter.Rewrite(host),
				Port:     rule.Backend.Service.Port,
				Request:  rule.Backend.Service.Request,
				Response: rule.Backend.Service.Response,
			}
			return s, &rule, nil
		}
	}

	// return nil, nil, fmt.Errorf("no rule found for host %s", host)
	return nil, nil, ErrHostNotFound
}

func matchPath(paths []rule.Path, path string) (r *service.Service, err error) {
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

	// return nil, fmt.Errorf("no rule found for path %s", path)
	return nil, ErrPathNotFound
}
