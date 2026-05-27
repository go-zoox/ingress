package waf

import "github.com/go-zoox/ingress/core/rule"

// BuiltinRuleEnabled reports whether a starter rule should run given global defaults and overrides.
// When builtin_rules[id] is set, it wins; otherwise enabled defaults to !disableBuiltin.
func BuiltinRuleEnabled(disableBuiltin bool, overrides map[string]bool, id string) bool {
	if overrides != nil {
		if v, ok := overrides[id]; ok {
			return v
		}
	}
	return !disableBuiltin
}

// RuleActive reports whether a custom or overridden signature rule should run.
// Omitted enabled means on; explicit enabled: false disables the rule.
func RuleActive(r rule.WAFRule) bool {
	if r.Enabled != nil {
		return *r.Enabled
	}
	return true
}

func filterStarterRules(merged rule.WAF) []rule.WAFRule {
	starters := StarterRules()
	out := make([]rule.WAFRule, 0, len(starters))
	for _, r := range starters {
		if BuiltinRuleEnabled(merged.DisableBuiltin, merged.BuiltinRules, r.ID) {
			out = append(out, r)
		}
	}
	return out
}
