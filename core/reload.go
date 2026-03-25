package core

import "fmt"

func (c *core) Reload(cfg *Config) error {
	c.cfg = cfg

	if err := c.prepare(); err != nil {
		return fmt.Errorf("failed to prepare: %s", err)
	}

	// prepare() -> prepareCache() clears the configured cache backend (same as startup).

	return nil
}
