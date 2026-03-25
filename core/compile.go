package core

import (
	"fmt"
	"regexp"

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

func compileRouterIndex(rules []rule.Rule, fallback rule.Backend) (*routerIndex, error) {
	idx := &routerIndex{
		entries:     make([]compiledRuleEntry, 0, len(rules)),
		pathsByRule: make([][]compiledPath, len(rules)),
		fallbackValid: fallback.Service.Name != "",
	}

	for i := range rules {
		r := &rules[i]
		switch r.HostType {
		case "exact", "":
			idx.entries = append(idx.entries, compiledRuleEntry{
				hostType:  "exact",
				ruleIndex: i,
				exactHost: r.Host,
			})
		case "regex":
			re, err := regexp.Compile(r.Host)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].host regex: %w", i, err)
			}
			idx.entries = append(idx.entries, compiledRuleEntry{
				hostType:  "regex",
				ruleIndex: i,
				re:        re,
			})
		case "wildcard":
			pat := wildCardToRegexp(r.Host)
			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, fmt.Errorf("rules[%d].host wildcard: %w", i, err)
			}
			idx.entries = append(idx.entries, compiledRuleEntry{
				hostType:             "wildcard",
				ruleIndex:            i,
				re:                   re,
				wildcardRewriterFrom: pat,
			})
		default:
			return nil, fmt.Errorf("unsupport host type: %s", r.HostType)
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
