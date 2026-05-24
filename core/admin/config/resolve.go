package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePaths makes relative admin paths absolute against the ingress config file directory.
func ResolvePaths(cfg *Config, ingressConfigFile string) error {
	if cfg == nil {
		return nil
	}
	base, err := ingressConfigDir(ingressConfigFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.IngressConfigPath) == "" {
		cfg.IngressConfigPath = ingressConfigFile
	}
	absConfig, err := absIngressConfigFile(cfg.IngressConfigPath)
	if err != nil {
		return err
	}
	cfg.IngressConfigPath = absConfig
	cfg.PidFile = resolveFilePath(base, cfg.PidFile)
	if strings.TrimSpace(cfg.AccessLogPath) != "" {
		cfg.AccessLogPath = resolveFilePath(base, cfg.AccessLogPath)
	}
	if strings.TrimSpace(cfg.ErrorLogPath) != "" {
		cfg.ErrorLogPath = resolveFilePath(base, cfg.ErrorLogPath)
	}
	cfg.Database.DSN = resolveSQLiteDSN(base, cfg.Database.DSN)
	if err := resolveIngressLogPaths(cfg); err != nil {
		return err
	}
	return nil
}

func ingressConfigDir(ingressConfigFile string) (string, error) {
	if strings.TrimSpace(ingressConfigFile) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return wd, nil
	}
	p := ingressConfigFile
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

// absIngressConfigFile resolves -c style paths against process cwd, not the config directory.
func absIngressConfigFile(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", os.ErrInvalid
	}
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		p = filepath.Join(wd, p)
	}
	return filepath.Abs(p)
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
