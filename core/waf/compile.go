package waf

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
)

type targetKind int

const (
	tkPath targetKind = iota
	tkQuery
	tkURI
	tkHeaders
	tkHeader
)

type sigRule struct {
	id             string
	name           string
	action         string
	actionExplicit bool
	ruleLogOnly    bool
	contains       bool
	pattern  string
	re       *regexp.Regexp
	targets  []targetKind
	hdrNames []string
}

// Profile is compiled WAF state for one ingress rule (or global fallback index -1 internally).
type Profile struct {
	ruleIndex int

	Enabled          bool // whether WAF runs for this profile
	GlobalLogOnly    bool
	TrustProxy       bool
	XFFIndex         int
	BlockStatus      int
	BlockContentType string
	BlockBody        string

	denyNet        []*net.IPNet
	allowNet       []*net.IPNet
	signatureRules []*sigRule
}

func compileProfile(ruleIndex int, mergedIn rule.WAF) (*Profile, error) {
	if !mergedIn.Enabled {
		return &Profile{ruleIndex: ruleIndex, Enabled: false}, nil
	}

	rLabel := fmt.Sprintf("rules[%d]", ruleIndex)
	if ruleIndex < 0 {
		rLabel = "waf(global)"
	}
	merged := mergedIn
	if acts, err := normalizeBuiltinRuleActions(merged.BuiltinRuleActions, rLabel); err != nil {
		return nil, err
	} else if acts != nil {
		merged.BuiltinRuleActions = acts
	}

	p := &Profile{
		ruleIndex:        ruleIndex,
		Enabled:          true,
		GlobalLogOnly:    merged.LogOnly,
		TrustProxy:       merged.TrustProxy,
		XFFIndex:         int(merged.XFFIndex),
		BlockStatus:      403,
		BlockContentType: "text/plain; charset=utf-8",
		BlockBody:        "Forbidden\n",
	}

	if merged.BlockStatusCode > 0 {
		p.BlockStatus = int(merged.BlockStatusCode)
	}
	if strings.TrimSpace(merged.BlockContentType) != "" {
		p.BlockContentType = merged.BlockContentType
	}
	if merged.BlockBody != "" {
		p.BlockBody = merged.BlockBody
	}

	ipPhaseLabel := rLabel

	for _, raw := range merged.Deny {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		ipnet, err := parseIPCIDR(raw)
		if err != nil {
			return nil, fmt.Errorf("%s.waf.deny[%q]: %w", ipPhaseLabel, raw, err)
		}
		p.denyNet = append(p.denyNet, ipnet)
	}

	for _, raw := range merged.Allow {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		ipnet, err := parseIPCIDR(raw)
		if err != nil {
			return nil, fmt.Errorf("%s.waf.allow[%q]: %w", ipPhaseLabel, raw, err)
		}
		p.allowNet = append(p.allowNet, ipnet)
	}

	custom := merged.Rules
	var combined []rule.WAFRule
	combined = append(combined, filterStarterRules(merged)...)
	combined = append(combined, custom...)

	seenSigID := make(map[string]struct{}, len(combined))

	for i := range combined {
		r := &combined[i]
		if !RuleActive(*r) {
			continue
		}
		if strings.TrimSpace(r.ID) == "" {
			return nil, fmt.Errorf("%s.waf.rules[%d]: id is required", rLabel, i)
		}
		if strings.TrimSpace(r.Pattern) == "" {
			return nil, fmt.Errorf("%s.waf.rules id=%q: pattern is required", rLabel, r.ID)
		}
		cr, err := compileSigRule(r)
		if err != nil {
			return nil, fmt.Errorf("%s.waf.rules id=%q: %w", rLabel, r.ID, err)
		}
		if _, dup := seenSigID[r.ID]; dup {
			return nil, fmt.Errorf("%s.waf.rules id=%q: duplicate id after merge", rLabel, r.ID)
		}
		seenSigID[r.ID] = struct{}{}
		p.signatureRules = append(p.signatureRules, cr)
	}

	return p, nil
}

func compileSigRule(r *rule.WAFRule) (*sigRule, error) {
	actionExplicit := strings.TrimSpace(r.Action) != ""
	action, err := normalizeRuleAction(r)
	if err != nil {
		return nil, err
	}
	cr := &sigRule{
		id:             r.ID,
		name:           r.Name,
		action:         action,
		actionExplicit: actionExplicit,
		ruleLogOnly:    r.LogOnly,
	}

	pt := strings.TrimSpace(strings.ToLower(r.Type))
	switch pt {
	case "", PatternTypeRegex:
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("regex: %w", err)
		}
		cr.re = re
		cr.contains = false
	case PatternTypeContains:
		cr.pattern = r.Pattern
		cr.contains = true
	default:
		return nil, fmt.Errorf("unsupported type %q (use %q or %q)", r.Type, PatternTypeRegex, PatternTypeContains)
	}

	if len(r.Targets) == 0 {
		return nil, fmt.Errorf("targets: at least one target is required")
	}

	for _, t := range r.Targets {
		t = strings.TrimSpace(strings.ToLower(t))
		switch t {
		case TargetPath:
			cr.targets = append(cr.targets, tkPath)
			cr.hdrNames = append(cr.hdrNames, "")
		case TargetQuery:
			cr.targets = append(cr.targets, tkQuery)
			cr.hdrNames = append(cr.hdrNames, "")
		case TargetURI:
			cr.targets = append(cr.targets, tkURI)
			cr.hdrNames = append(cr.hdrNames, "")
		case TargetHeaders:
			cr.targets = append(cr.targets, tkHeaders)
			cr.hdrNames = append(cr.hdrNames, "")
		case "":
			return nil, fmt.Errorf("targets: empty entry")
		default:
			if strings.HasPrefix(t, headerPrefix) {
				name := strings.TrimSpace(t[len(headerPrefix):])
				if name == "" {
					return nil, fmt.Errorf("targets: empty header name in %q", t)
				}
				cr.targets = append(cr.targets, tkHeader)
				cr.hdrNames = append(cr.hdrNames, strings.ToLower(name))
				continue
			}
			return nil, fmt.Errorf("targets: unknown target %q", t)
		}
	}

	return cr, nil
}

func parseIPCIDR(s string) (*net.IPNet, error) {
	if strings.Contains(s, "/") {
		_, ipnet, err := net.ParseCIDR(s)
		return ipnet, err
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP %q", s)
	}
	if v4 := ip.To4(); v4 != nil {
		return &net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}, nil
	}
	v6 := ip.To16()
	return &net.IPNet{IP: v6, Mask: net.CIDRMask(128, 128)}, nil
}

// CompileIngress builds a Profile per ingress rule plus a global-merge-only fallback Profile.
func CompileIngress(global rule.WAF, rules []rule.Rule) ([]*Profile, *Profile, error) {
	perRule := make([]*Profile, len(rules))
	for i := range rules {
		merged, err := MergePatch(global, rules[i].WAFPatch, i)
		if err != nil {
			return nil, nil, err
		}
		p, err := compileProfile(i, merged)
		if err != nil {
			return nil, nil, err
		}
		perRule[i] = p
	}
	fbMerge, err := MergePatch(global, nil, -1)
	if err != nil {
		return nil, nil, err
	}
	fb, err := compileProfile(-1, fbMerge)
	if err != nil {
		return nil, nil, err
	}
	return perRule, fb, nil
}

func ipMatchesNets(ip net.IP, nets []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n != nil && n.Contains(ip) {
			return true
		}
	}
	return false
}
