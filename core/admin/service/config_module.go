package service

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigModule is one editable section of ingress.yaml for the admin UI.
type ConfigModule struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Keys  []string `json:"keys"`
	YAML  string   `json:"yaml"`
}

type configModuleDef struct {
	ID    string
	Label string
	Keys  []string
}

// deprecatedConfigKeys are ignored by ingress core; stripped on module split/merge (legacy YAML).
var deprecatedConfigKeys = map[string]struct{}{
	"version": {},
}

var configModuleDefs = []configModuleDef{
	{ID: "general", Label: "基础", Keys: []string{"port", "enable_h2c", "error_page_expose_details", "error_pages"}},
	{ID: "services", Label: "服务", Keys: []string{"services"}},
	{ID: "rules", Label: "路由规则", Keys: []string{"rules"}},
	{ID: "admin", Label: "Admin 控制台", Keys: []string{"admin"}},
	{ID: "cache", Label: "缓存", Keys: []string{"cache"}},
	{ID: "logging", Label: "日志", Keys: []string{"logging"}},
	{ID: "waf", Label: "WAF", Keys: []string{"waf"}},
	{ID: "maintenance", Label: "维护", Keys: []string{"maintenance"}},
	{ID: "rate_limit", Label: "限流", Keys: []string{"rate_limit"}},
	{ID: "security", Label: "安全", Keys: []string{"security"}},
	{ID: "healthcheck", Label: "健康检查", Keys: []string{"healthcheck"}},
	{ID: "https", Label: "HTTPS", Keys: []string{"https"}},
	{ID: "fallback", Label: "Fallback", Keys: []string{"fallback"}},
	{ID: "scenarios", Label: "场景", Keys: []string{"scenarios"}},
	{ID: "jobs", Label: "定时任务", Keys: []string{"jobs"}},
}

func moduleKeysSet() map[string]string {
	out := make(map[string]string)
	for _, def := range configModuleDefs {
		for _, key := range def.Keys {
			out[key] = def.ID
		}
	}
	return out
}

func SplitConfigModules(content string) ([]ConfigModule, error) {
	root, err := parseYAMLRootMapping(content)
	if err != nil {
		return nil, err
	}

	keyNodes := rootMappingIndex(root)
	assigned := make(map[string]bool)
	modules := make([]ConfigModule, 0, len(configModuleDefs)+1)

	for _, def := range configModuleDefs {
		pairs := make([]*yaml.Node, 0, len(def.Keys)*2)
		for _, key := range def.Keys {
			if node, ok := keyNodes[key]; ok {
				pairs = append(pairs, scalarNode(key), node)
				assigned[key] = true
			}
		}
		modules = append(modules, ConfigModule{
			ID:    def.ID,
			Label: def.Label,
			Keys:  append([]string(nil), def.Keys...),
			YAML:  encodeMappingPairs(pairs),
		})
	}

	otherPairs := make([]*yaml.Node, 0)
	for i := 0; i < len(root.Content); i += 2 {
		key := strings.TrimSpace(root.Content[i].Value)
		if assigned[key] {
			continue
		}
		if _, legacy := deprecatedConfigKeys[key]; legacy {
			assigned[key] = true
			continue
		}
		otherPairs = append(otherPairs, root.Content[i], root.Content[i+1])
	}
	// "其他" only when there are top-level keys outside known modules (edit in UI or YAML tab).
	if len(otherPairs) > 0 {
		modules = append(modules, ConfigModule{
			ID:    "other",
			Label: "其他",
			Keys:  otherKeys(otherPairs),
			YAML:  encodeMappingPairs(otherPairs),
		})
	}

	return modules, nil
}

func MergeConfigModule(content, moduleID, moduleYAML string) (string, error) {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return "", fmt.Errorf("module_id is required")
	}

	root, err := parseYAMLRootMapping(content)
	if err != nil {
		return "", err
	}

	removeKeys := moduleRemovalKeys(moduleID, root)
	if moduleID != "other" && len(removeKeys) == 0 {
		return "", fmt.Errorf("unknown module %q", moduleID)
	}

	moduleYAML = strings.TrimSpace(moduleYAML)
	var patchRoot *yaml.Node
	if moduleYAML != "" {
		patchRoot, err = parseYAMLRootMapping(moduleYAML)
		if err != nil {
			return "", fmt.Errorf("module yaml: %w", err)
		}
	}

	removed := make(map[string]bool, len(removeKeys))
	for _, key := range removeKeys {
		removed[key] = true
	}

	if patchRoot != nil {
		known := moduleKeysSet()
		patchPairs := make([]*yaml.Node, 0, len(patchRoot.Content))
		for i := 0; i < len(patchRoot.Content); i += 2 {
			key := strings.TrimSpace(patchRoot.Content[i].Value)
			if moduleID == "other" {
				if _, isKnown := known[key]; isKnown {
					continue
				}
			} else if !moduleContainsKey(moduleID, key) {
				continue
			}
			patchPairs = append(patchPairs, patchRoot.Content[i], patchRoot.Content[i+1])
		}

		next := make([]*yaml.Node, 0, len(root.Content)+len(patchPairs))
		inserted := false
		for i := 0; i < len(root.Content); i += 2 {
			key := strings.TrimSpace(root.Content[i].Value)
			if removed[key] {
				if !inserted && len(patchPairs) > 0 {
					next = append(next, patchPairs...)
					inserted = true
				}
				continue
			}
			next = append(next, root.Content[i], root.Content[i+1])
		}
		if !inserted && len(patchPairs) > 0 {
			next = append(next, patchPairs...)
		}
		root.Content = next
	} else {
		next := make([]*yaml.Node, 0, len(root.Content))
		for i := 0; i < len(root.Content); i += 2 {
			key := strings.TrimSpace(root.Content[i].Value)
			if removed[key] {
				continue
			}
			next = append(next, root.Content[i], root.Content[i+1])
		}
		root.Content = next
	}

	return encodeMappingNode(root), nil
}

func ChangedConfigModules(baseline, draft string) ([]string, error) {
	baseMods, err := SplitConfigModules(baseline)
	if err != nil {
		return nil, err
	}
	draftMods, err := SplitConfigModules(draft)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]string, len(baseMods))
	for _, m := range baseMods {
		byID[m.ID] = normalizeModuleYAML(m.YAML)
	}
	changed := make([]string, 0)
	for _, m := range draftMods {
		if byID[m.ID] != normalizeModuleYAML(m.YAML) {
			changed = append(changed, m.ID)
		}
	}
	return changed, nil
}

func moduleRemovalKeys(moduleID string, root *yaml.Node) []string {
	if moduleID == "other" {
		known := moduleKeysSet()
		keys := make([]string, 0)
		for i := 0; i < len(root.Content); i += 2 {
			key := strings.TrimSpace(root.Content[i].Value)
			if _, ok := known[key]; !ok {
				keys = append(keys, key)
			}
		}
		return keys
	}
	for _, def := range configModuleDefs {
		if def.ID == moduleID {
			keys := append([]string(nil), def.Keys...)
			if moduleID == "general" {
				for k := range deprecatedConfigKeys {
					keys = append(keys, k)
				}
			}
			return keys
		}
	}
	return nil
}

func moduleContainsKey(moduleID, key string) bool {
	for _, def := range configModuleDefs {
		if def.ID != moduleID {
			continue
		}
		for _, k := range def.Keys {
			if k == key {
				return true
			}
		}
	}
	return false
}

func normalizeModuleYAML(y string) string {
	y = strings.TrimSpace(y)
	if y == "" {
		return ""
	}
	root, err := parseYAMLRootMapping(y)
	if err != nil {
		return y
	}
	return encodeMappingNode(root)
}

func parseYAMLRootMapping(content string) (*yaml.Node, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		doc := &yaml.Node{Kind: yaml.DocumentNode}
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		return doc.Content[0], nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("yaml syntax error: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty yaml document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("ingress config root must be a mapping")
	}
	return root, nil
}

func rootMappingIndex(root *yaml.Node) map[string]*yaml.Node {
	out := make(map[string]*yaml.Node)
	if root == nil || root.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i < len(root.Content); i += 2 {
		out[strings.TrimSpace(root.Content[i].Value)] = root.Content[i+1]
	}
	return out
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func encodeMappingPairs(pairs []*yaml.Node) string {
	if len(pairs) == 0 {
		return ""
	}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: pairs}
	return encodeMappingNode(root)
}

func encodeMappingNode(root *yaml.Node) string {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	_ = enc.Encode(doc)
	_ = enc.Close()
	return strings.TrimRight(buf.String(), "\n")
}

func otherKeys(pairs []*yaml.Node) []string {
	if len(pairs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		keys = append(keys, strings.TrimSpace(pairs[i].Value))
	}
	return keys
}
