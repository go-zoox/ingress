package waf

import (
	"fmt"
	"os"

	"github.com/go-zoox/ingress/core/rule"
	"gopkg.in/yaml.v3"
)

// ApplyRulePatchesFromFile reads YAML and sets rules[i].WAFPatch from each rules[].waf map.
func ApplyRulePatchesFromFile(path string, rules []rule.Rule) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ApplyRulePatchesFromYAML(b, rules)
}

// ApplyRulePatchesFromYAML extracts rules[].waf mapping nodes (post config.Load).
func ApplyRulePatchesFromYAML(data []byte, rules []rule.Rule) error {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}
	yr, _ := root["rules"].([]any)
	for i := range rules {
		rules[i].WAFPatch = nil
		if i >= len(yr) {
			continue
		}
		rmap, ok := yr[i].(map[string]any)
		if !ok {
			continue
		}
		w, ok := rmap["waf"]
		if !ok || w == nil {
			continue
		}
		patch, ok := w.(map[string]any)
		if !ok {
			return fmt.Errorf("rules[%d].waf must be a mapping", i)
		}
		rules[i].WAFPatch = patch
	}
	return nil
}
