package waf

import (
	"strings"
	"testing"

	"github.com/go-zoox/ingress/core/rule"
)

func TestMergePatch_PartialKeepsInherited(t *testing.T) {
	t.Parallel()
	base := rule.WAF{Enabled: true, Deny: []string{"10.0.0.1"}, DisableBuiltin: true}
	out, err := MergePatch(base, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Deny) != 1 {
		t.Fatalf("deny: %+v", out.Deny)
	}
	out2, err := MergePatch(base, map[string]any{"trust_proxy": true}, 1)
	if err != nil || !out2.Enabled || len(out2.Deny) != 1 || !out2.TrustProxy {
		t.Fatalf("unexpected merge: %+v err=%v", out2, err)
	}
}

func TestMergeRules_ByID(t *testing.T) {
	t.Parallel()
	g := []rule.WAFRule{
		{ID: "a", Pattern: "."},
		{ID: "b", Pattern: "."},
	}
	o := []rule.WAFRule{
		{ID: "b", Pattern: "(?i)zzz"},
		{ID: "c", Pattern: "."},
	}
	out := MergeRules(g, o)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	for _, r := range out {
		if r.ID == "b" && r.Pattern != "(?i)zzz" {
			t.Fatalf("overlay did not replace b: %q", r.Pattern)
		}
	}
}

func TestMergePatch_RulesFromPatch_MergesByID(t *testing.T) {
	t.Parallel()
	base := rule.WAF{
		Enabled:        true,
		DisableBuiltin: true,
		Rules: []rule.WAFRule{
			{ID: "a", Type: PatternTypeContains, Pattern: "1", Targets: []string{TargetPath}},
		},
	}
	patch := map[string]any{
		"rules": []any{
			map[string]any{
				"id":      "a",
				"type":    PatternTypeContains,
				"pattern": "2",
				"targets": []any{"path"},
			},
			map[string]any{
				"id":      "b",
				"type":    PatternTypeContains,
				"pattern": "3",
				"targets": []any{"path"},
			},
		},
	}
	out, err := MergePatch(base, patch, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Rules) != 2 {
		t.Fatalf("rules: %+v", out.Rules)
	}
	if out.Rules[0].Pattern != "2" {
		t.Fatalf("overlay a: %q", out.Rules[0].Pattern)
	}
	if out.Rules[1].ID != "b" {
		t.Fatalf("append b: %+v", out.Rules[1])
	}
}

func TestMergePatch_InvalidBool(t *testing.T) {
	t.Parallel()
	_, err := MergePatch(rule.WAF{}, map[string]any{"enabled": "maybe"}, 0)
	if err == nil {
		t.Fatal("expected invalid bool")
	}
}

func TestMergePatch_Global_IndexNegative_AllKeys_and_QualPrefixes(t *testing.T) {
	t.Parallel()
	base := rule.WAF{
		Enabled:          false,
		TrustProxy:       false,
		XFFIndex:         9,
		LogOnly:          true,
		BlockStatusCode:  403,
		BlockContentType: "old",
		BlockBody:        "oldbody",
		DisableBuiltin:   false,
		Deny:             []string{"10.0.0.10"},
		Allow:            []string{"10.11.11.11"},
		Rules:            []rule.WAFRule{{ID: "seed", Pattern: ".", Targets: []string{TargetPath}}},
	}
	out, err := MergePatch(base, map[string]any{
		"enabled":            "on",
		"trust_proxy":        true,
		"xff_index":          int32(-2),
		"log_only":           "no",
		"block_status_code":  float64(418),
		"block_content_type": "application/x-test",
		"block_body":         "boom",
		"disable_builtin":    "true",
		"deny":               nil,
		"allow":              []any{"172.26.26.26"},
		"rules":              nil,
	}, -1)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Enabled || !out.TrustProxy || out.XFFIndex != -2 || out.LogOnly {
		t.Fatalf("parsed flags: %+v", out)
	}
	if out.BlockStatusCode != 418 || out.BlockContentType != "application/x-test" || out.BlockBody != "boom" {
		t.Fatalf("block fields: %+v", out)
	}
	if !out.DisableBuiltin || out.Deny != nil || len(out.Allow) != 1 {
		t.Fatalf("lists: deny=%v allow=%v", out.Deny, out.Allow)
	}
	if len(out.Rules) != 1 || out.Rules[0].ID != "seed" {
		t.Fatalf("rules nil patch should preserve base rules slice: %+v", out.Rules)
	}

	_, err = MergePatch(rule.WAF{}, map[string]any{"enabled": []any{}}, 0)
	if err == nil || !strings.Contains(err.Error(), "rules[0].waf.enabled") {
		t.Fatalf("expected enabled type error with rules[] prefix: %v", err)
	}
}

func TestMergePatch_AsInt64_variants(t *testing.T) {
	t.Parallel()
	outi, err := MergePatch(rule.WAF{}, map[string]any{"xff_index": int(42)}, 0)
	if err != nil || outi.XFFIndex != 42 {
		t.Fatalf("xff int: %v %+v", err, outi)
	}

	outStr, err := MergePatch(rule.WAF{}, map[string]any{"block_status_code": "500"}, -1)
	if err != nil || outStr.BlockStatusCode != 500 {
		t.Fatalf("block_status string int: %v %+v", err, outStr)
	}

	_, err = MergePatch(rule.WAF{}, map[string]any{"enabled": []any{}, "trust_proxy": true}, 0)
	if err == nil || !strings.Contains(err.Error(), "enabled") {
		t.Fatalf("enabled wrong type via separate patch: %v", err)
	}

	_, err = MergePatch(rule.WAF{}, map[string]any{"xff_index": map[string]any{}}, -1)
	if err == nil || !strings.Contains(err.Error(), "expected integer") {
		t.Fatalf("xff_index wrong type err=%v", err)
	}
}

func TestMergePatch_AsBool_stringTruthiness(t *testing.T) {
	t.Parallel()
	wantTrue := []string{"true", "TRUE", "1", "yes", "On"}
	for _, s := range wantTrue {
		out, err := MergePatch(rule.WAF{}, map[string]any{"trust_proxy": s}, -1)
		if err != nil || !out.TrustProxy {
			t.Fatalf("%q trust_proxy=%v err=%v", s, out.TrustProxy, err)
		}
	}
	for _, s := range []string{"false", "0", "no", "off"} {
		out, err := MergePatch(rule.WAF{TrustProxy: true}, map[string]any{"trust_proxy": s}, -1)
		if err != nil || out.TrustProxy {
			t.Fatalf("%q trust_proxy=%v err=%v", s, out.TrustProxy, err)
		}
	}
}

func TestMergePatch_Deny_allow_strSlice_errors(t *testing.T) {
	t.Parallel()

	_, err := MergePatch(rule.WAF{}, map[string]any{"deny": "one"}, -1)
	if err == nil || !strings.Contains(err.Error(), "deny") || !strings.Contains(err.Error(), "array") {
		t.Fatalf("deny not slice: %v", err)
	}

	_, err = MergePatch(rule.WAF{}, map[string]any{"allow": []any{"ok", 42}}, -1)
	if err == nil || !strings.Contains(err.Error(), "allow[1]: expected string") {
		t.Fatalf("allow elt type: %v", err)
	}
}

func TestMergePatch_patchRules_slice_errors_and_rule_map(t *testing.T) {
	t.Parallel()

	_, err := MergePatch(rule.WAF{}, map[string]any{"rules": "oops"}, -1)
	if err == nil || !strings.Contains(err.Error(), "rules: expected array") {
		t.Fatalf("rules wrong type: %v", err)
	}

	_, err = MergePatch(rule.WAF{}, map[string]any{"rules": []any{42}}, -1)
	if err == nil || !strings.Contains(err.Error(), "[0]: expected mapping") {
		t.Fatalf("rules[0]: %v", err)
	}

	_, err = MergePatch(rule.WAF{}, map[string]any{"rules": []any{map[string]any{"id": 3}}}, -1)
	if err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("id non-string: %v", err)
	}

	out, err := MergePatch(rule.WAF{}, map[string]any{"rules": []any{map[string]any{
		"id": "nr", "name": []any{}, "pattern": "x",
	}}}, -1)
	if err == nil || !strings.Contains(err.Error(), ".name") {
		t.Fatalf("name not string: err=%v out=%+v", err, out)
	}
}

func TestMergePatch_AsString_strict(t *testing.T) {
	t.Parallel()
	_, err := MergePatch(rule.WAF{}, map[string]any{"block_content_type": 12}, -1)
	if err == nil || !strings.Contains(err.Error(), "block_content_type") {
		t.Fatalf("expected string err: %v", err)
	}
}

func TestMergeRules_skipEmptyPatchID(t *testing.T) {
	t.Parallel()
	got := MergeRules(nil, []rule.WAFRule{
		{ID: "", Pattern: "ignored"},
		{ID: "keep", Pattern: "x"},
	})
	if len(got) != 1 || got[0].ID != "keep" {
		t.Fatalf("%+v", got)
	}
}
