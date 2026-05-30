package core

import (
	"os"

	"github.com/go-zoox/ingress/core/waf"
)

// FinalizeLoadedConfig resolves paths, normalizes logging, validates/applies scenarios, and applies WAF rule patches.
// When yamlBytes is nil, WAF patches are read from configFilePath.
func FinalizeLoadedConfig(cfg *Config, configFilePath string, yamlBytes []byte) error {
	if yamlBytes == nil && configFilePath != "" {
		b, err := os.ReadFile(configFilePath)
		if err != nil {
			return err
		}
		yamlBytes = b
	}
	if len(yamlBytes) > 0 {
		if err := enrichScenariosFromYAML(yamlBytes, cfg); err != nil {
			return err
		}
	}
	if err := ResolveConfigPaths(cfg, configFilePath); err != nil {
		return err
	}
	if err := cfg.Logging.Normalize(); err != nil {
		return err
	}
	if err := ValidateScenariosConfig(cfg); err != nil {
		return err
	}
	if err := ApplyScenarios(cfg); err != nil {
		return err
	}
	if len(yamlBytes) > 0 {
		if err := waf.ApplyRulePatchesFromYAML(yamlBytes, cfg.Rules); err != nil {
			return err
		}
	}
	return nil
}
