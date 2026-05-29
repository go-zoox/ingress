package waf

import (
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestCompileProfile_InvalidAction(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true,
		Rules: []rule.WAFRule{{
			ID:      "bad",
			Action:  "drop",
			Type:    PatternTypeContains,
			Pattern: "x",
			Targets: []string{TargetPath},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported action") {
		t.Fatalf("want unsupported action error, got %v", err)
	}
}

func TestCompileProfile_InvalidBuiltinRuleAction(t *testing.T) {
	t.Parallel()
	_, err := compileProfile(0, rule.WAF{
		Enabled: true,
		BuiltinRuleActions: map[string]string{
			"builtin:xss-lite": "invalid",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "builtin_rule_actions") {
		t.Fatalf("want builtin_rule_actions error, got %v", err)
	}
}
