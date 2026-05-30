package core

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func enrichScenariosFromYAML(yamlBytes []byte, cfg *Config) error {
	if !cfg.Scenarios.Configured() {
		return nil
	}
	var root map[string]any
	if err := yaml.Unmarshal(yamlBytes, &root); err != nil {
		return fmt.Errorf("scenarios yaml: %w", err)
	}
	scenariosVal, ok := root["scenarios"]
	if !ok {
		return nil
	}
	scenariosMap, ok := scenariosVal.(map[string]any)
	if !ok {
		return fmt.Errorf("scenarios must be a mapping")
	}
	itemsVal, ok := scenariosMap["items"]
	if !ok {
		return nil
	}
	itemsList, ok := itemsVal.([]any)
	if !ok {
		return fmt.Errorf("scenarios.items must be a sequence")
	}
	if len(itemsList) != len(cfg.Scenarios.Items) {
		return fmt.Errorf("scenarios.items length mismatch after load (yaml=%d config=%d)", len(itemsList), len(cfg.Scenarios.Items))
	}
	overlays := make(map[string]map[string]any, len(cfg.Scenarios.Items))
	for i, item := range cfg.Scenarios.Items {
		id := strings.TrimSpace(item.ID)
		entry, ok := itemsList[i].(map[string]any)
		if !ok {
			return fmt.Errorf("scenarios.items[%d] must be a mapping", i)
		}
		if entryID, _ := entry["id"].(string); strings.TrimSpace(entryID) != id {
			return fmt.Errorf("scenarios.items[%d]: id %q does not match loaded config id %q", i, entryID, id)
		}
		if rawOverlay, ok := entry["overlay"]; ok && rawOverlay != nil {
			overlay, ok := rawOverlay.(map[string]any)
			if !ok {
				return fmt.Errorf("scenarios.items[%d].overlay must be a mapping", i)
			}
			overlays[id] = overlay
		}
	}
	cfg.Scenarios.overlays = overlays
	return nil
}
