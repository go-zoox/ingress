package core

import (
	"fmt"
	"os"
)

func (c *core) Reload(cfg *Config) error {
	c.cfg = cfg

	if err := c.prepare(); err != nil {
		return fmt.Errorf("failed to prepare: %s", err)
	}

	c.refreshRuntimeConfigHash()

	// prepare() -> prepareCache() clears the configured cache backend (same as startup).

	return nil
}

func (c *core) ConfigFingerprint() string {
	return c.runtimeConfigHash
}

func (c *core) refreshRuntimeConfigHash() {
	if c.configFilePath != "" {
		b, err := os.ReadFile(c.configFilePath)
		if err == nil {
			c.runtimeConfigHash = ContentHash(string(b))
			return
		}
	}
	c.runtimeConfigHash = ""
}
