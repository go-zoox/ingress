package service

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaintenanceHostEntry is one maintenance host pattern with an optional time window.
type MaintenanceHostEntry struct {
	Host   string            `config:"host"`
	Window MaintenanceWindow `config:"window"`
}

func (e MaintenanceHostEntry) Pattern() string {
	return strings.TrimSpace(e.Host)
}

// MaintenanceHostList is maintenance.hosts; accepts plain strings or {host, window} mappings.
type MaintenanceHostList []MaintenanceHostEntry

func (l MaintenanceHostList) Patterns() []string {
	out := make([]string, 0, len(l))
	for _, e := range l {
		if p := e.Pattern(); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// UnmarshalYAML supports:
//   - app.example.com
//   - host: app.example.com
//     window: { start: "...", end: "..." }
func (l *MaintenanceHostList) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind == yaml.ScalarNode && value.Value == "" {
		*l = nil
		return nil
	}
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("hosts must be a sequence")
	}
	out := make(MaintenanceHostList, 0, len(value.Content))
	for i, item := range value.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			host := strings.TrimSpace(item.Value)
			if host == "" {
				continue
			}
			out = append(out, MaintenanceHostEntry{Host: host})
		case yaml.MappingNode:
			var entry MaintenanceHostEntry
			if err := item.Decode(&entry); err != nil {
				return fmt.Errorf("hosts[%d]: %w", i, err)
			}
			if entry.Pattern() == "" {
				return fmt.Errorf("hosts[%d].host is required", i)
			}
			out = append(out, entry)
		default:
			return fmt.Errorf("hosts[%d]: must be a string or mapping", i)
		}
	}
	*l = out
	return nil
}
