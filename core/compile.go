package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

// routerIndex holds precompiled host/path matchers for cfg.Rules (iteration order preserved).
type routerIndex struct {
	entries       []compiledRuleEntry
	pathsByRule   [][]compiledPath
	fallbackValid bool
}

type compiledRuleEntry struct {
	hostType             string // exact | regex | wildcard
	ruleIndex            int
	exactHost            string
	re                   *regexp.Regexp
	wildcardRewriterFrom string
}

type compiledPath struct {
	re        *regexp.Regexp
	pathIndex int
}

// effectiveHostType resolves the matcher kind used at runtime. When declared is empty
// or "auto", regex is preferred over wildcard if host contains regexp metacharacters
// (so patterns like ^.*\.example\.com$ are not treated as wildcards).
func effectiveHostType(declared, host string) string {
	switch declared {
	case hostTypeRegex, hostTypeWildcard, hostTypeExact:
		return declared
	case "", hostTypeAuto:
		if hostLooksLikeRegexp(host) {
			return hostTypeRegex
		}
		if strings.Contains(host, "*") {
			return hostTypeWildcard
		}
		return hostTypeExact
	default:
		return declared
	}
}

func hostLooksLikeRegexp(host string) bool {
	for _, r := range host {
		switch r {
		case '(', ')', '[', ']', '^', '$', '|', '+', '?', '\\':
			return true
		}
	}
	return false
}

func compileRouterIndex(rules []rule.Rule, fallback rule.Backend) (*routerIndex, error) {
	idx := &routerIndex{
		entries:       make([]compiledRuleEntry, 0, len(rules)),
		pathsByRule:   make([][]compiledPath, len(rules)),
		fallbackValid: fallback.Service.Name != "",
	}

	for i := range rules {
		r := &rules[i]
		ht := effectiveHostType(r.HostType, r.Host)
		r.HostType = ht
		switch ht {
		case hostTypeExact:
			idx.entries = append(idx.entries, compiledRuleEntry{
				hostType:  hostTypeExact,
				ruleIndex: i,
				exactHost: r.Host,
			})
		case hostTypeRegex:
			re, err := regexp.Compile(r.Host)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].host regex: %w", i, err)
			}
			idx.entries = append(idx.entries, compiledRuleEntry{
				hostType:  hostTypeRegex,
				ruleIndex: i,
				re:        re,
			})
		case hostTypeWildcard:
			pat := wildCardToRegexp(r.Host)
			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].host wildcard: %w", i, err)
			}
			idx.entries = append(idx.entries, compiledRuleEntry{
				hostType:             hostTypeWildcard,
				ruleIndex:            i,
				re:                   re,
				wildcardRewriterFrom: pat,
			})
		default:
			return nil, fmt.Errorf("unsupport host type: %s", ht)
		}

		for j := range r.Paths {
			p := &r.Paths[j]
			re, err := regexp.Compile("^" + p.Path)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].paths[%d]: %w", i, j, err)
			}
			idx.pathsByRule[i] = append(idx.pathsByRule[i], compiledPath{re: re, pathIndex: j})
		}
	}

	return idx, nil
}
