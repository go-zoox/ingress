package main

import (
	"fmt"

	"github.com/go-zoox/ingress/core"
	adminapp "github.com/go-zoox/ingress/core/admin/app"
	admincfg "github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/logger"
)

func startAdmin(ingressApp core.Core, ingressCfg *core.Config, configFilePath, pidFilePath string) error {
	adminCfg, err := buildAdminConfig(ingressApp, ingressCfg, configFilePath, pidFilePath, ingressApp.ReloadFromFile)
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

func buildAdminConfig(ingressApp core.Core, ingressCfg *core.Config, ingressConfigFile, pidFile string, reloadFn func() error) (*admincfg.Config, error) {
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
		GeoIP: admincfg.GeoIP{
			Database:     a.GeoIP.Database,
			IngressLat:   a.GeoIP.IngressLat,
			IngressLng:   a.GeoIP.IngressLng,
			IngressLabel: a.GeoIP.IngressLabel,
		},
		Auth: admincfg.Auth{
			Type: a.Auth.Type,
			Basic: admincfg.AuthBasic{
				Username: a.Auth.Basic.Username,
				Password: a.Auth.Basic.Password,
			},
			OAuth: admincfg.AuthOAuth{
				Provider:     a.Auth.OAuth.Provider,
				ClientID:     a.Auth.OAuth.ClientID,
				ClientSecret: a.Auth.OAuth.ClientSecret,
				RedirectURL:  a.Auth.OAuth.RedirectURL,
				Scopes:       append([]string(nil), a.Auth.OAuth.Scopes...),
			},
		},
		IngressConfigPath: ingressConfigFile,
		PidFile:           pidFile,
		ReloadFn:          reloadFn,
		CoreInstance:      ingressApp,
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
