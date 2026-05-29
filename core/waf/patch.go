package waf

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

// MergeRules combines base-ordered rules with patch rules by stable id (patch replaces same id).
func MergeRules(base, patch []rule.WAFRule) []rule.WAFRule {
	byID := make(map[string]rule.WAFRule, len(patch))
	order := make([]string, 0, len(patch))
	for _, r := range patch {
		if r.ID == "" {
			continue
		}
		if _, seen := byID[r.ID]; !seen {
			order = append(order, r.ID)
		}
		byID[r.ID] = r
	}
	inBase := make(map[string]struct{}, len(base))

	out := make([]rule.WAFRule, 0, len(base)+len(patch))
	for _, g := range base {
		if g.ID != "" {
			inBase[g.ID] = struct{}{}
		}
		if r, ok := byID[g.ID]; ok && g.ID != "" {
			out = append(out, r)
			continue
		}
		out = append(out, g)
	}

	for _, id := range order {
		if _, old := inBase[id]; old {
			continue
		}
		out = append(out, byID[id])
	}
	return out
}

// MergePatch overlays rules[].waf YAML map keys onto typed global baseline rule.WAF.
func MergePatch(base rule.WAF, patch map[string]any, ruleIdx int) (rule.WAF, error) {
	out := base
	rLoc := ""
	if ruleIdx >= 0 {
		rLoc = fmt.Sprintf("rules[%d].waf", ruleIdx)
	}

	if len(patch) == 0 {
		return out, nil
	}

	if v, ok := patch["enabled"]; ok {
		b, err := asBool(v, qual(rLoc, "enabled"))
		if err != nil {
			return out, err
		}
		out.Enabled = b
	}
	if v, ok := patch["trust_proxy"]; ok {
		b, err := asBool(v, qual(rLoc, "trust_proxy"))
		if err != nil {
			return out, err
		}
		out.TrustProxy = b
	}
	if v, ok := patch["xff_index"]; ok {
		x, err := asInt64(v, qual(rLoc, "xff_index"))
		if err != nil {
			return out, err
		}
		out.XFFIndex = x
	}
	if v, ok := patch["log_only"]; ok {
		b, err := asBool(v, qual(rLoc, "log_only"))
		if err != nil {
			return out, err
		}
		out.LogOnly = b
	}
	if v, ok := patch["block_status_code"]; ok {
		x, err := asInt64(v, qual(rLoc, "block_status_code"))
		if err != nil {
			return out, err
		}
		out.BlockStatusCode = x
	}
	if v, ok := patch["block_content_type"]; ok {
		s, err := asString(v, qual(rLoc, "block_content_type"))
		if err != nil {
			return out, err
		}
		out.BlockContentType = s
	}
	if v, ok := patch["block_body"]; ok {
		s, err := asString(v, qual(rLoc, "block_body"))
		if err != nil {
			return out, err
		}
		out.BlockBody = s
	}
	if v, ok := patch["disable_builtin"]; ok {
		b, err := asBool(v, qual(rLoc, "disable_builtin"))
		if err != nil {
			return out, err
		}
		out.DisableBuiltin = b
	}
	if v, ok := patch["builtin_rules"]; ok {
		m, err := boolMap(v, qual(rLoc, "builtin_rules"))
		if err != nil {
			return out, err
		}
		if len(m) > 0 {
			if out.BuiltinRules == nil {
				out.BuiltinRules = make(map[string]bool, len(m))
			}
			for k, val := range m {
				out.BuiltinRules[k] = val
			}
		}
	}
	if v, ok := patch["builtin_rule_actions"]; ok {
		m, err := stringMap(v, qual(rLoc, "builtin_rule_actions"))
		if err != nil {
			return out, err
		}
		if len(m) > 0 {
			if out.BuiltinRuleActions == nil {
				out.BuiltinRuleActions = make(map[string]string, len(m))
			}
			for k, val := range m {
				norm, err := NormalizeAction(val, false)
				if err != nil {
					return out, fmt.Errorf("%s[%q]: %w", qual(rLoc, "builtin_rule_actions"), k, err)
				}
				out.BuiltinRuleActions[k] = norm
			}
		}
	}
	if v, ok := patch["deny"]; ok {
		sl, err := strSlice(v, qual(rLoc, "deny"))
		if err != nil {
			return out, err
		}
		out.Deny = sl
	}
	if v, ok := patch["allow"]; ok {
		sl, err := strSlice(v, qual(rLoc, "allow"))
		if err != nil {
			return out, err
		}
		out.Allow = sl
	}
	if v, ok := patch["allow_hosts"]; ok {
		sl, err := strSlice(v, qual(rLoc, "allow_hosts"))
		if err != nil {
			return out, err
		}
		out.AllowHosts = sl
	}
	if v, ok := patch["rules"]; ok {
		parsed, err := patchRulesSlice(v, qual(rLoc, "rules"))
		if err != nil {
			return out, err
		}
		out.Rules = MergeRules(out.Rules, parsed)
	}

	return out, nil
}

func qual(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func asBool(v any, ctx string) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		s := strings.TrimSpace(strings.ToLower(x))
		switch s {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return false, fmt.Errorf("%s: invalid bool %q", ctx, x)
		}
	default:
		return false, fmt.Errorf("%s: expected bool", ctx)
	}
}

func asInt64(v any, ctx string) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case float64:
		return int64(x), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	default:
		return 0, fmt.Errorf("%s: expected integer", ctx)
	}
}

func asString(v any, ctx string) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	default:
		return "", fmt.Errorf("%s: expected string", ctx)
	}
}

func strSlice(v any, ctx string) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected array of strings", ctx)
	}
	out := make([]string, len(arr))
	for i, elt := range arr {
		switch s := elt.(type) {
		case string:
			out[i] = s
		default:
			return nil, fmt.Errorf("%s[%d]: expected string", ctx, i)
		}
	}
	return out, nil
}

func patchRulesSlice(v any, ctx string) ([]rule.WAFRule, error) {
	if v == nil {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected array", ctx)
	}
	out := make([]rule.WAFRule, len(arr))
	for i, elt := range arr {
		m, ok := elt.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s[%d]: expected mapping", ctx, i)
		}
		r, err := patchRuleFromMap(m, fmt.Sprintf("%s[%d]", ctx, i))
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

func stringMap(v any, ctx string) (map[string]string, error) {
	if v == nil {
		return nil, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected mapping", ctx)
	}
	out := make(map[string]string, len(m))
	for k, elt := range m {
		key := strings.TrimSpace(k)
		if key == "" {
			return nil, fmt.Errorf("%s: empty key", ctx)
		}
		s, err := asString(elt, fmt.Sprintf("%s[%q]", ctx, key))
		if err != nil {
			return nil, err
		}
		out[key] = s
	}
	return out, nil
}

func boolMap(v any, ctx string) (map[string]bool, error) {
	if v == nil {
		return nil, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected mapping", ctx)
	}
	out := make(map[string]bool, len(m))
	for k, elt := range m {
		key := strings.TrimSpace(k)
		if key == "" {
			return nil, fmt.Errorf("%s: empty key", ctx)
		}
		b, err := asBool(elt, fmt.Sprintf("%s[%q]", ctx, key))
		if err != nil {
			return nil, err
		}
		out[key] = b
	}
	return out, nil
}

func patchRuleFromMap(m map[string]any, ctx string) (rule.WAFRule, error) {
	var r rule.WAFRule

	if v, ok := m["id"]; ok {
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return r, fmt.Errorf("%s.id: non-empty string required", ctx)
		}
		r.ID = strings.TrimSpace(s)
	}

	if v, ok := m["name"]; ok {
		s, err := asString(v, ctx+".name")
		if err != nil {
			return r, err
		}
		r.Name = s
	}

	if v, ok := m["log_only"]; ok {
		b, err := asBool(v, ctx+".log_only")
		if err != nil {
			return r, err
		}
		r.LogOnly = b
	}

	if v, ok := m["action"]; ok {
		s, err := asString(v, ctx+".action")
		if err != nil {
			return r, err
		}
		norm, err := NormalizeAction(s, r.LogOnly)
		if err != nil {
			return r, fmt.Errorf("%s.action: %w", ctx, err)
		}
		r.Action = norm
	}

	if v, ok := m["enabled"]; ok {
		b, err := asBool(v, ctx+".enabled")
		if err != nil {
			return r, err
		}
		r.Enabled = &b
	}

	if v, ok := m["type"]; ok {
		s, err := asString(v, ctx+".type")
		if err != nil {
			return r, err
		}
		r.Type = s
	}

	if v, ok := m["pattern"]; ok {
		s, err := asString(v, ctx+".pattern")
		if err != nil {
			return r, err
		}
		r.Pattern = s
	}

	if v, ok := m["targets"]; ok {
		sl, err := strSlice(v, ctx+".targets")
		if err != nil {
			return r, err
		}
		r.Targets = sl
	}

	if v, ok := m["allow_hosts"]; ok {
		sl, err := strSlice(v, ctx+".allow_hosts")
		if err != nil {
			return r, err
		}
		r.AllowHosts = sl
	}

	return r, nil
}
