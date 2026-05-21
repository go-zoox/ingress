package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePaths makes relative ingress paths absolute against the admin config file directory.
func ResolvePaths(cfg *Config, adminConfigFile string) error {
	if cfg == nil {
		return nil
	}
	base, err := adminConfigDir(adminConfigFile)
	if err != nil {
		return err
	}
	cfg.Ingress.ConfigPath = resolveFilePath(base, cfg.Ingress.ConfigPath)
	cfg.Ingress.PidFile = resolveFilePath(base, cfg.Ingress.PidFile)
	if strings.TrimSpace(cfg.Ingress.LogPath) != "" {
		cfg.Ingress.LogPath = resolveFilePath(base, cfg.Ingress.LogPath)
	}
	if strings.TrimSpace(cfg.Ingress.ErrorLogPath) != "" {
		cfg.Ingress.ErrorLogPath = resolveFilePath(base, cfg.Ingress.ErrorLogPath)
	}
	cfg.Database.DSN = resolveSQLiteDSN(base, cfg.Database.DSN)
	if err := resolveIngressLogPaths(cfg); err != nil {
		return err
	}
	return nil
}

func adminConfigDir(adminConfigFile string) (string, error) {
	if strings.TrimSpace(adminConfigFile) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return wd, nil
	}
	p := adminConfigFile
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		p = filepath.Join(wd, p)
	}
	return filepath.Abs(filepath.Dir(p))
}

func resolveFilePath(base, p string) string {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(base, p))
}

func resolveSQLiteDSN(base, dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if !strings.HasPrefix(dsn, "file:") {
		return dsn
	}
	rest := strings.TrimPrefix(dsn, "file:")
	path, query, hasQuery := strings.Cut(rest, "?")
	if filepath.IsAbs(path) {
		return dsn
	}
	abs := filepath.Clean(filepath.Join(base, path))
	if hasQuery {
		return "file:" + abs + "?" + query
	}
	return "file:" + abs
}
