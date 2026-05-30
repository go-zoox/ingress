package core

import (
	"fmt"

	"github.com/go-zoox/config"
)

func (c *core) ReloadFromFile() error {
	if c.configFilePath == "" {
		return fmt.Errorf("ingress config path is not set")
	}
	var cfg Config
	if err := config.Load(&cfg, &config.LoadOptions{FilePath: c.configFilePath}); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := FinalizeLoadedConfig(&cfg, c.configFilePath, nil); err != nil {
		return fmt.Errorf("prepare config: %w", err)
	}
	if err := c.Reload(&cfg); err != nil {
		return err
	}
	return nil
}
