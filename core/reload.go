package core

import "fmt"

func (c *core) Reload(cfg *Config) error {
	c.cfg = cfg

	if err := c.prepare(); err != nil {
		return fmt.Errorf("failed to prepare: %s", err)
	}

	// @TODO clear cache

	return nil
}
