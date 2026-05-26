package core

import (
	"fmt"

	"github.com/go-zoox/config"
	"github.com/go-zoox/ingress/core/waf"
)

func (c *core) ReloadFromFile() error {
	if c.configFilePath == "" {
		return fmt.Errorf("ingress config path is not set")
	}
	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: c.configFilePath}); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := ResolveConfigPaths(&cfg, c.configFilePath); err != nil {
		return fmt.Errorf("resolve paths: %w", err)
	}
	if err := waf.ApplyRulePatchesFromFile(c.configFilePath, cfg.Rules); err != nil {
		return fmt.Errorf("waf patches: %w", err)
	}
	if err := c.Reload(&cfg); err != nil {
		return err
	}
	return nil
}
