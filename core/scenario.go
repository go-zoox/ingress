package core

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-zoox/ingress/core/rule"
	"gopkg.in/yaml.v3"
)

const envScenarioOverride = "INGRESS_SCENARIO"

// DefaultScenarioID is the reserved active value: use root ingress.yaml as-is (no overlay merge).
const DefaultScenarioID = "default"

var scenarioOverlayTopLevelKeys = map[string]struct{}{
	"cache":       {},
	"rate_limit":  {},
	"waf":         {},
	"maintenance": {},
	"security":    {},
	"rules":       {},
}

// ScenarioItem is one selectable scenario; overlay is parsed from raw YAML (see enrichScenariosFromYAML).
type ScenarioItem struct {
	ID          string `config:"id"`
	Label       string `config:"label"`
	Description string `config:"description"`
}

// Scenarios configures named runtime overlays (方案 C: active + items[]).
type Scenarios struct {
	Active   string                    `config:"active"`
	Items    []ScenarioItem            `config:"items"`
	overlays map[string]map[string]any `config:"-"`
}

// ScenarioSummary is a list entry for Admin Console.
type ScenarioSummary struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

// ScenariosResponse is returned by GET /api/v1/scenarios.
type ScenariosResponse struct {
	Active    string            `json:"active"`
	Scenarios []ScenarioSummary `json:"scenarios"`
}

// IsDefaultScenario reports whether id selects the root config without overlay.
func IsDefaultScenario(id string) bool {
	return strings.TrimSpace(id) == "" || strings.TrimSpace(id) == DefaultScenarioID
}

// Configured reports whether any scenario items are defined.
func (s Scenarios) Configured() bool {
	return len(s.Items) > 0
}

// ValidateScenariosConfig checks scenario metadata before overlay merge.
func ValidateScenariosConfig(cfg *Config) error {
	if !cfg.Scenarios.Configured() {
		return nil
	}
	seen := make(map[string]struct{}, len(cfg.Scenarios.Items))
	for i, item := range cfg.Scenarios.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return fmt.Errorf("scenarios.items[%d].id is required", i)
		}
		if id == DefaultScenarioID {
			return fmt.Errorf("scenarios.items[%d].id %q is reserved (use scenarios.active: default for root config)", i, id)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("scenarios.items[%d]: duplicate id %q", i, id)
		}
		seen[id] = struct{}{}
		if overlay, ok := cfg.Scenarios.overlays[id]; ok {
			if err := validateScenarioOverlay(overlay, i); err != nil {
				return err
			}
		}
	}
	active := strings.TrimSpace(cfg.Scenarios.Active)
	if active == "" {
		active = DefaultScenarioID
	}
	if !IsDefaultScenario(active) {
		if _, ok := seen[active]; !ok {
			return fmt.Errorf("scenarios.active %q is not defined in scenarios.items", active)
		}
	}
	return nil
}

func validateScenarioOverlay(overlay map[string]any, itemIdx int) error {
	if len(overlay) == 0 {
		return nil
	}
	for key := range overlay {
		if _, ok := scenarioOverlayTopLevelKeys[key]; !ok {
			return fmt.Errorf("scenarios.items[%d].overlay: unsupported key %q (allowed: cache, rate_limit, waf, maintenance, security, rules)", itemIdx, key)
		}
	}
	if rulesVal, ok := overlay["rules"]; ok && rulesVal != nil {
		patches, err := asRuleOverlayMaps(rulesVal)
		if err != nil {
			return fmt.Errorf("scenarios.items[%d].overlay.rules: %w", itemIdx, err)
		}
		for j, patch := range patches {
			host, _ := patch["host"].(string)
			if strings.TrimSpace(host) == "" {
				return fmt.Errorf("scenarios.items[%d].overlay.rules[%d].host is required", itemIdx, j)
			}
		}
	}
	return nil
}

// ResolveActiveScenario returns the effective scenario id (env override wins).
func ResolveActiveScenario(cfg *Config) string {
	if v := strings.TrimSpace(os.Getenv(envScenarioOverride)); v != "" {
		return v
	}
	return strings.TrimSpace(cfg.Scenarios.Active)
}

// ApplyScenarios merges the active scenario overlay into cfg (in place).
func ApplyScenarios(cfg *Config) error {
	if !cfg.Scenarios.Configured() {
		return nil
	}
	active := ResolveActiveScenario(cfg)
	if IsDefaultScenario(active) {
		return nil
	}
	var item *ScenarioItem
	for i := range cfg.Scenarios.Items {
		if cfg.Scenarios.Items[i].ID == active {
			item = &cfg.Scenarios.Items[i]
			break
		}
	}
	if item == nil {
		return fmt.Errorf("scenarios.active %q is not defined in scenarios.items", active)
	}
	overlay := cfg.Scenarios.overlays[active]
	return applyScenarioOverlay(cfg, overlay)
}

func applyScenarioOverlay(cfg *Config, overlay map[string]any) error {
	if len(overlay) == 0 {
		return nil
	}
	for key, val := range overlay {
		if val == nil {
			continue
		}
		switch key {
		case "cache":
			if err := mergeSection(&cfg.Cache, val); err != nil {
				return fmt.Errorf("scenarios overlay cache: %w", err)
			}
		case "rate_limit":
			if err := mergeSection(&cfg.RateLimit, val); err != nil {
				return fmt.Errorf("scenarios overlay rate_limit: %w", err)
			}
		case "waf":
			if err := mergeSection(&cfg.WAF, val); err != nil {
				return fmt.Errorf("scenarios overlay waf: %w", err)
			}
		case "maintenance":
			if err := mergeSection(&cfg.Maintenance, val); err != nil {
				return fmt.Errorf("scenarios overlay maintenance: %w", err)
			}
		case "security":
			if err := mergeSection(&cfg.Security, val); err != nil {
				return fmt.Errorf("scenarios overlay security: %w", err)
			}
		case "rules":
			patches, err := asRuleOverlayMaps(val)
			if err != nil {
				return fmt.Errorf("scenarios overlay rules: %w", err)
			}
			merged, err := mergeRulesByHostMaps(cfg.Rules, patches)
			if err != nil {
				return err
			}
			cfg.Rules = merged
		default:
			return fmt.Errorf("scenarios overlay: unsupported key %q", key)
		}
	}
	return nil
}

// ListScenarios builds Admin list metadata from cfg (before or after apply; active uses ResolveActiveScenario).
func ListScenarios(cfg *Config) ScenariosResponse {
	active := ResolveActiveScenario(cfg)
	if active == "" {
		active = DefaultScenarioID
	}
	out := ScenariosResponse{
		Active: active,
		Scenarios: []ScenarioSummary{
			{
				ID:          DefaultScenarioID,
				Label:       "默认",
				Description: "根配置，不应用 overlay",
				Active:      IsDefaultScenario(active),
			},
		},
	}
	for _, item := range cfg.Scenarios.Items {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = item.ID
		}
		out.Scenarios = append(out.Scenarios, ScenarioSummary{
			ID:          item.ID,
			Label:       label,
			Description: item.Description,
			Active:      item.ID == active,
		})
	}
	return out
}

func mergeRulesByHostMaps(base []rule.Rule, patches []map[string]any) ([]rule.Rule, error) {
	if len(patches) == 0 {
		return base, nil
	}
	out := append([]rule.Rule(nil), base...)
	for _, patch := range patches {
		host, _ := patch["host"].(string)
		host = strings.TrimSpace(host)
		if host == "" {
			return nil, fmt.Errorf("scenarios overlay rules: host is required")
		}
		idx := ruleIndexByExactHost(out, host)
		if idx >= 0 {
			merged, err := mergeRulePatchMap(out[idx], patch)
			if err != nil {
				return nil, fmt.Errorf("scenarios overlay rules host %q: %w", host, err)
			}
			out[idx] = merged
			continue
		}
		inserted, err := ruleFromOverlayPatch(patch)
		if err != nil {
			return nil, fmt.Errorf("scenarios overlay rules host %q: %w", host, err)
		}
		insertAt := ruleInsertIndexBeforeMatch(out, host)
		out = insertRuleAt(out, insertAt, inserted)
	}
	return out, nil
}

func ruleIndexByExactHost(rules []rule.Rule, host string) int {
	for i := range rules {
		if rules[i].Host == host {
			return i
		}
	}
	return -1
}

// ruleInsertIndexBeforeMatch returns the index before the first base rule that would match host at runtime.
// When no rule matches, append at len(rules).
func ruleInsertIndexBeforeMatch(rules []rule.Rule, host string) int {
	for i := range rules {
		if ruleHostMatches(rules[i], host) {
			return i
		}
	}
	return len(rules)
}

func ruleHostMatches(r rule.Rule, host string) bool {
	ht := effectiveHostType(r.HostType, r.Host)
	switch ht {
	case hostTypeExact:
		return r.Host == host
	case hostTypeRegex:
		re, err := regexp.Compile(r.Host)
		if err != nil {
			return false
		}
		return re.MatchString(host)
	case hostTypeWildcard:
		re, err := regexp.Compile(wildCardToRegexp(r.Host))
		if err != nil {
			return false
		}
		return re.MatchString(host)
	default:
		return false
	}
}

func ruleFromOverlayPatch(patch map[string]any) (rule.Rule, error) {
	raw, err := yaml.Marshal(patch)
	if err != nil {
		return rule.Rule{}, err
	}
	var r rule.Rule
	if err := yaml.Unmarshal(raw, &r); err != nil {
		return rule.Rule{}, err
	}
	return r, nil
}

func insertRuleAt(rules []rule.Rule, idx int, r rule.Rule) []rule.Rule {
	if idx < 0 {
		idx = 0
	}
	if idx > len(rules) {
		idx = len(rules)
	}
	out := make([]rule.Rule, 0, len(rules)+1)
	out = append(out, rules[:idx]...)
	out = append(out, r)
	out = append(out, rules[idx:]...)
	return out
}

func mergeRulePatchMap(base rule.Rule, patch map[string]any) (rule.Rule, error) {
	var out rule.Rule
	if err := mergeSection(&base, patch); err != nil {
		return out, err
	}
	return base, nil
}

func mergeSection(dst, src any) error {
	dstBytes, err := yaml.Marshal(dst)
	if err != nil {
		return err
	}
	srcBytes, err := yaml.Marshal(src)
	if err != nil {
		return err
	}
	var dstMap, srcMap map[string]any
	if err := yaml.Unmarshal(dstBytes, &dstMap); err != nil {
		return err
	}
	if err := yaml.Unmarshal(srcBytes, &srcMap); err != nil {
		return err
	}
	deepMergeMap(dstMap, srcMap)
	merged, err := yaml.Marshal(dstMap)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(merged, dst)
}

func deepMergeMap(dst, src map[string]any) {
	for k, v := range src {
		if v == nil {
			continue
		}
		if existing, ok := dst[k]; ok {
			dstChild, dstOK := existing.(map[string]any)
			srcChild, srcOK := v.(map[string]any)
			if dstOK && srcOK {
				deepMergeMap(dstChild, srcChild)
				continue
			}
		}
		dst[k] = v
	}
}

func asRuleOverlayMaps(v any) ([]map[string]any, error) {
	raw, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
