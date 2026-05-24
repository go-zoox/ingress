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
	access := strings.TrimSpace(cfg.AccessLogPath)
	errorLog := strings.TrimSpace(cfg.ErrorLogPath)
	if access != "" && errorLog != "" {
		return nil
	}

	fromIngressAccess, fromIngressError := logPathsFromIngressConfig(cfg.IngressConfigPath)
	if access == "" {
		if fromIngressAccess != "" {
			access = fromIngressAccess
		} else {
			access = ingcore.DefaultAccessLogPath
		}
		cfg.AccessLogPath = access
	}
	if errorLog == "" {
		if fromIngressError != "" {
			errorLog = fromIngressError
		} else {
			errorLog = ingcore.DefaultErrorLogPath
		}
		cfg.ErrorLogPath = errorLog
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
	_ = cfg.Logging.Prepare(cfg.Admin, configPath)
	return cfg.Logging.FileLogPaths()
}
