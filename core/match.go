package core

import (
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
	for _, rule := range rules {
		hostRegExp := fmt.Sprintf("^%s$", rule.Host)
		if isMatched, _ := regexp.MatchString(hostRegExp, host); isMatched {
			// paths
			if rule.Paths != nil {
				for _, rpath := range rule.Paths {
					rpathRe := fmt.Sprintf("^%s", rpath.Path)
					//
					isMatched, err := regexp.MatchString(rpathRe, path)
					if err != nil {
						return nil, fmt.Errorf("failed to match path: %s", err)
					}

					if isMatched {
						s = &service.Service{
							Protocol: rpath.Backend.Service.Protocol,
							Name:     rpath.Backend.Service.Name,
							Port:     rpath.Backend.Service.Port,
							Request:  rpath.Backend.Service.Request,
							Response: rpath.Backend.Service.Response,
						}
						break
					}
				}
			}

			// main
			if s == nil {
				hostRewriter := rewriter.Rewriter{
					From: hostRegExp,
					To:   rule.Backend.Service.Name,
				}
				serviceNameNew := hostRewriter.Rewrite(host)
				s = &service.Service{
					Protocol: rule.Backend.Service.Protocol,
					Name:     serviceNameNew,
					Port:     rule.Backend.Service.Port,
					Request:  rule.Backend.Service.Request,
					Response: rule.Backend.Service.Response,
				}
			}
		}
	}

	return s, nil
}
