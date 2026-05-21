package config

import (
	"strings"

	zcfg "github.com/go-zoox/config"
	"github.com/go-zoox/fs"
	ingcore "github.com/go-zoox/ingress/core"
)

func resolveIngressLogPaths(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	access := strings.TrimSpace(cfg.Ingress.LogPath)
	errorLog := strings.TrimSpace(cfg.Ingress.ErrorLogPath)
	if access != "" && errorLog != "" {
		return nil
	}

	fromIngressAccess, fromIngressError := logPathsFromIngressConfig(cfg.Ingress.ConfigPath)
	if access == "" {
		if fromIngressAccess != "" {
			access = fromIngressAccess
		} else {
			access = ingcore.DefaultAccessLogPath
		}
		cfg.Ingress.LogPath = access
	}
	if errorLog == "" {
		if fromIngressError != "" {
			errorLog = fromIngressError
		} else {
			errorLog = ingcore.DefaultErrorLogPath
		}
		cfg.Ingress.ErrorLogPath = errorLog
	}
	return nil
}

func logPathsFromIngressConfig(configPath string) (access, errorLog string) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" || !fs.IsExist(configPath) {
		return "", ""
	}
	var cfg ingcore.Config
	if err := zcfg.Load(&cfg, &zcfg.LoadOptions{FilePath: configPath}); err != nil {
		return "", ""
	}
	if err := ingcore.ResolveConfigPaths(&cfg, configPath); err != nil {
		return "", ""
	}
	_ = cfg.Logging.Normalize()
	return cfg.Logging.FileLogPaths()
}
