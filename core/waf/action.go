package waf

import (
	"fmt"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

// Rule actions for signature hits and built-in overrides.
const (
	ActionBlock = "block"
	ActionAudit = "audit"
	ActionPass  = "pass"
)

// NormalizeAction maps YAML/config values to a canonical action.
// Empty action with log_only true becomes audit; empty otherwise block.
func NormalizeAction(action string, logOnly bool) (string, error) {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		if logOnly {
			return ActionAudit, nil
		}
		return ActionBlock, nil
	}
	switch action {
	case ActionBlock, ActionAudit, ActionPass:
		return action, nil
	case "log", "log_only":
		return ActionAudit, nil
	case "allow":
		return ActionPass, nil
	default:
		return "", fmt.Errorf("unsupported action %q (use block, audit, or pass)", action)
	}
}

func normalizeRuleAction(r *rule.WAFRule) (string, error) {
	return NormalizeAction(r.Action, r.LogOnly)
}

func normalizeBuiltinRuleActions(m map[string]string, label string) (map[string]string, error) {
	if len(m) == 0 {
		return m, nil
	}
	out := make(map[string]string, len(m))
	for id, act := range m {
		norm, err := NormalizeAction(act, false)
		if err != nil {
			return nil, fmt.Errorf("%s.waf.builtin_rule_actions[%q]: %w", label, id, err)
		}
		out[id] = norm
	}
	return out, nil
}

// resolveSigAction applies explicit per-rule action, rule log_only, then global log_only.
func resolveSigAction(sr *sigRule, globalLogOnly bool) string {
	if sr.actionExplicit {
		return sr.action
	}
	if sr.ruleLogOnly || globalLogOnly {
		return ActionAudit
	}
	return sr.action
}
