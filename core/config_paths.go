package core

import (
	"os"
	"path/filepath"
	"strings"

	zcfg "github.com/go-zoox/zoox/config"
)

// ResolveConfigPaths makes relative paths in cfg absolute against the ingress
// config file directory (same rule as relative paths in ingress.yaml).
func ResolveConfigPaths(cfg *Config, configFilePath string) error {
	if cfg == nil {
		return nil
	}
	base, err := ingressConfigDir(configFilePath)
	if err != nil {
		return err
	}
	resolveLoggingPaths(&cfg.Logging, base)
	resolveAdminPaths(&cfg.Admin, base)
	resolveHTTPSPaths(&cfg.HTTPS, base)
	return nil
}

func resolveAdminPaths(admin *Admin, base string) {
	if admin == nil {
		return
	}
	admin.AccessLogPath = resolveConfigFilePath(base, admin.AccessLogPath)
	admin.ErrorLogPath = resolveConfigFilePath(base, admin.ErrorLogPath)
}

func ingressConfigDir(configFilePath string) (string, error) {
	if strings.TrimSpace(configFilePath) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return wd, nil
	}
	p := configFilePath
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		p = filepath.Join(wd, p)
	}
	return filepath.Abs(filepath.Dir(p))
}

func resolveLoggingPaths(l *Logging, base string) {
	if l == nil {
		return
	}
	for i := range l.Transports {
		resolveTransportPaths(&l.Transports[i], base)
	}
}

func resolveTransportPaths(t *zcfg.Transport, base string) {
	if t == nil {
		return
	}
	if typ := strings.ToLower(strings.TrimSpace(t.Type)); typ != "" && typ != "file" {
		return
	}
	t.Path = resolveConfigFilePath(base, t.Path)
	if len(t.Levels) == 0 {
		return
	}
	for level, path := range t.Levels {
		t.Levels[level] = resolveConfigFilePath(base, path)
	}
}

func resolveConfigFilePath(base, p string) string {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(base, p))
}

func resolveHTTPSPaths(https *HTTPS, base string) {
	if https == nil {
		return
	}
	for i := range https.SSL {
		https.SSL[i].Cert.Certificate = resolveConfigFilePath(base, https.SSL[i].Cert.Certificate)
		https.SSL[i].Cert.CertificateKey = resolveConfigFilePath(base, https.SSL[i].Cert.CertificateKey)
	}
}
