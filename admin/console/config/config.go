package config

import (
	"fmt"
	"strings"
)

// Config is the ingress admin server configuration.
type Config struct {
	Port int64 `config:"port,default=9080"`
	//
	Ingress Ingress `config:"ingress"`
	//
	Database Database `config:"database"`
	//
	Web Web `config:"web"`
}

type Ingress struct {
	ConfigPath   string `config:"config_path,default=/etc/ingress/config.yaml"`
	PidFile      string `config:"pid_file,default=/tmp/gozoox.ingress.pid"`
	LogPath      string `config:"log_path"`
	ErrorLogPath string `config:"error_log_path"`
}

type Database struct {
	Driver string `config:"driver,default=sqlite"`
	DSN    string `config:"dsn,default=file:admin.db?cache=shared&_fk=1"`
}

type Web struct {
	// DevProxy when true serves API only; frontend runs on Vite dev server.
	DevProxy bool `config:"dev_proxy"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(c.Ingress.ConfigPath) == "" {
		return fmt.Errorf("ingress.config_path is required")
	}
	d := strings.ToLower(strings.TrimSpace(c.Database.Driver))
	switch d {
	case "sqlite", "sqlite3", "mysql", "postgres", "postgresql", "":
	default:
		return fmt.Errorf("unsupported database.driver %q", c.Database.Driver)
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
