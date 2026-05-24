package main

import (
	"fmt"

	"github.com/go-zoox/ingress/core"
	adminapp "github.com/go-zoox/ingress/core/admin/app"
	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/logger"
)

func startAdmin(ingressApp core.Core, ingressCfg *core.Config, configFilePath, pidFilePath string) error {
	adminCfg, err := buildAdminConfig(ingressCfg, configFilePath, pidFilePath, ingressApp.ReloadFromFile)
	if err != nil {
		return fmt.Errorf("admin: %w", err)
	}
	adminApp, err := adminapp.New(adminCfg)
	if err != nil {
		return fmt.Errorf("admin: %w", err)
	}
	go func() {
		if err := adminapp.Run(adminApp, adminCfg); err != nil {
			logger.Errorf("admin server stopped: %s", err)
		}
	}()
	return nil
}

func buildAdminConfig(ingressCfg *core.Config, ingressConfigFile, pidFile string, reloadFn func() error) (*admincfg.Config, error) {
	a := ingressCfg.Admin
	accessLog, errorLog := "", ""
	if ingressCfg != nil {
		accessLog, errorLog = ingressCfg.Logging.FileLogPaths()
	}
	cfg := &admincfg.Config{
		Enabled:           a.Enabled,
		Port:              a.Port,
		Database:          admincfg.Database{Driver: a.Database.Driver, DSN: a.Database.DSN},
		Web:               admincfg.Web{DevProxy: a.Web.DevProxy},
		AccessLogPath:     a.AccessLogPath,
		ErrorLogPath:      a.ErrorLogPath,
		IngressConfigPath: ingressConfigFile,
		PidFile:           pidFile,
		ReloadFn:          reloadFn,
	}
	if cfg.AccessLogPath == "" {
		cfg.AccessLogPath = accessLog
	}
	if cfg.ErrorLogPath == "" {
		cfg.ErrorLogPath = errorLog
	}
	if cfg.Port == 0 {
		cfg.Port = 9080
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = "file:admin.db?cache=shared&_fk=1"
	}
	if err := admincfg.ResolvePaths(cfg, ingressConfigFile); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
