package waf

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

// combineSignatureRules merges starter rules with custom waf.rules[] entries by id.
// Custom entries overlay same-id starters (pattern, targets, allow_hosts, etc.) and append new ids.
func combineSignatureRules(merged rule.WAF) []rule.WAFRule {
	starters := filterStarterRules(merged)
	byID := make(map[string]rule.WAFRule, len(starters)+len(merged.Rules))
	order := make([]string, 0, len(starters)+len(merged.Rules))

	for _, r := range starters {
		id := strings.TrimSpace(r.ID)
		if id == "" {
			continue
		}
		byID[id] = r
		order = append(order, id)
	}
	for _, r := range merged.Rules {
		id := strings.TrimSpace(r.ID)
		if id == "" {
			continue
		}
		if existing, ok := byID[id]; ok {
			byID[id] = overlayWAFRule(existing, r)
			continue
		}
		byID[id] = r
		order = append(order, id)
	}

	out := make([]rule.WAFRule, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func overlayWAFRule(base, patch rule.WAFRule) rule.WAFRule {
	out := base
	if strings.TrimSpace(patch.Name) != "" {
		out.Name = patch.Name
	}
	if strings.TrimSpace(patch.Action) != "" {
		out.Action = patch.Action
		out.LogOnly = patch.LogOnly
	} else if patch.LogOnly {
		out.LogOnly = patch.LogOnly
	}
	if patch.Enabled != nil {
		out.Enabled = patch.Enabled
	}
	if strings.TrimSpace(patch.Type) != "" {
		out.Type = patch.Type
	}
	if strings.TrimSpace(patch.Pattern) != "" {
		out.Pattern = patch.Pattern
	}
	if len(patch.Targets) > 0 {
		out.Targets = patch.Targets
	}
	if len(patch.AllowHosts) > 0 {
		out.AllowHosts = patch.AllowHosts
	}
	return out
}

func validateCustomWAFRules(merged rule.WAF, rLabel string) error {
	seen := make(map[string]int)
	for i, r := range merged.Rules {
		id := strings.TrimSpace(r.ID)
		if id == "" {
			return fmt.Errorf("%s.waf.rules[%d]: id is required", rLabel, i)
		}
		if prev, dup := seen[id]; dup {
			return fmt.Errorf("%s.waf.rules[%d]: duplicate id %q (also at index %d)", rLabel, i, id, prev)
		}
		seen[id] = i
	}
	return nil
}
