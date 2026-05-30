package core

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SetScenariosActiveYAML updates scenarios.active in ingress YAML content.
func SetScenariosActiveYAML(content, activeID string) (string, error) {
	activeID = strings.TrimSpace(activeID)
	if activeID == "" {
		return "", fmt.Errorf("scenario id is required")
	}
	root, err := parseYAMLRootMapping(content)
	if err != nil {
		return "", err
	}
	scenariosNode, ok := rootMappingGet(root, "scenarios")
	if !ok || scenariosNode == nil || scenariosNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("scenarios block is not configured")
	}
	if !scenarioIDExists(scenariosNode, activeID) && activeID != DefaultScenarioID {
		return "", fmt.Errorf("scenario id %q is not defined in scenarios.items", activeID)
	}
	setMappingScalar(scenariosNode, "active", activeID)
	return encodeYAMLRootMapping(root), nil
}

func scenarioIDExists(scenariosNode *yaml.Node, id string) bool {
	itemsNode, ok := mappingNodeGet(scenariosNode, "items")
	if !ok || itemsNode == nil || itemsNode.Kind != yaml.SequenceNode {
		return false
	}
	for _, item := range itemsNode.Content {
		if item == nil || item.Kind != yaml.MappingNode {
			continue
		}
		idNode, ok := mappingNodeGet(item, "id")
		if ok && strings.TrimSpace(idNode.Value) == id {
			return true
		}
	}
	return false
}

func parseYAMLRootMapping(content string) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("yaml syntax error: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty yaml document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("root must be a mapping")
	}
	return root, nil
}

func rootMappingGet(root *yaml.Node, key string) (*yaml.Node, bool) {
	return mappingNodeGet(root, key)
}

func mappingNodeGet(m *yaml.Node, key string) (*yaml.Node, bool) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, false
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if strings.TrimSpace(m.Content[i].Value) == key {
			return m.Content[i+1], true
		}
	}
	return nil, false
}

func setMappingScalar(m *yaml.Node, key, value string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if strings.TrimSpace(m.Content[i].Value) == key {
			m.Content[i+1] = scalarYAMLNode(value)
			return
		}
	}
	m.Content = append(m.Content, scalarYAMLNode(key), scalarYAMLNode(value))
}

func scalarYAMLNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func encodeYAMLRootMapping(root *yaml.Node) string {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	_ = enc.Encode(root)
	_ = enc.Close()
	return strings.TrimSpace(buf.String()) + "\n"
}
