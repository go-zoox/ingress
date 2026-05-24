package config

import (
	"fmt"
	"strings"
)

// Config is the resolved admin console settings for the running ingress process.
type Config struct {
	Enabled      bool
	Port         int64
	Database     Database
	Web          Web
	AccessLogPath string
	ErrorLogPath string
	//
	IngressConfigPath string
	PidFile           string
	ReloadFn          func() error
}

type Database struct {
	Driver string
	DSN    string
}

type Web struct {
	// DevProxy when true serves API only; frontend runs on Vite dev server.
	DevProxy bool
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if !c.Enabled {
		return nil
	}
	if c.Port <= 0 {
		return fmt.Errorf("admin.port must be positive")
	}
	if strings.TrimSpace(c.IngressConfigPath) == "" {
		return fmt.Errorf("ingress config path is required")
	}
	d := strings.ToLower(strings.TrimSpace(c.Database.Driver))
	switch d {
	case "sqlite", "sqlite3", "mysql", "postgres", "postgresql", "":
	default:
		return fmt.Errorf("unsupported admin.database.driver %q", c.Database.Driver)
	}
	return nil
}

func (c *Config) DatabaseEngine() string {
	switch strings.ToLower(strings.TrimSpace(c.Database.Driver)) {
	case "mysql":
		return "mysql"
	case "postgres", "postgresql":
		return "postgres"
	default:
		return "sqlite"
	}
}
