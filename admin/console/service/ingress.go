package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	zcfg "github.com/go-zoox/config"
	"github.com/go-zoox/fs"
	admincfg "github.com/go-zoox/ingress/admin/console/config"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/waf"
	"gopkg.in/yaml.v3"
)

// Ingress wraps ingress config file operations.
type Ingress struct {
	cfg *admincfg.Config
}

func NewIngress(cfg *admincfg.Config) *Ingress {
	return &Ingress{cfg: cfg}
}

func (s *Ingress) ConfigPath() string {
	return s.cfg.Ingress.ConfigPath
}

func (s *Ingress) ReadYAML() (string, error) {
	path := s.ConfigPath()
	if !fs.IsExist(path) {
		return "", fmt.Errorf("config file(%s) not found", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Ingress) WriteYAML(content string) error {
	path := s.ConfigPath()
	return os.WriteFile(path, []byte(content), 0o644)
}

func (s *Ingress) LoadConfig() (*ingcore.Config, error) {
	path := s.ConfigPath()
	if !fs.IsExist(path) {
		return nil, fmt.Errorf("config file(%s) not found", path)
	}
	var cfg ingcore.Config
	if err := zcfg.Load(&cfg, &zcfg.LoadOptions{FilePath: path}); err != nil {
		return nil, err
	}
	if err := waf.ApplyRulePatchesFromFile(path, cfg.Rules); err != nil {
		return nil, err
	}
	if err := ingcore.ResolveConfigPaths(&cfg, path); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Ingress) ValidateYAML(content string) error {
	var node map[string]any
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		return fmt.Errorf("yaml syntax error: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.ConfigPath()), ".ingress-validate-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write([]byte(content)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	var cfg ingcore.Config
	if err := zcfg.Load(&cfg, &zcfg.LoadOptions{FilePath: tmpPath}); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := waf.ApplyRulePatchesFromYAML([]byte(content), cfg.Rules); err != nil {
		return err
	}
	return ingcore.ValidateConfig(&cfg)
}

func (s *Ingress) ValidateFile() error {
	content, err := s.ReadYAML()
	if err != nil {
		return err
	}
	return s.ValidateYAML(content)
}

func ingressReloadHint(configPath, pidFile string) string {
	return fmt.Sprintf(
		"start ingress in another terminal: ingress run -c %s --pid-file %s",
		configPath,
		pidFile,
	)
}

// ReloadReady reports whether SIGHUP reload can be sent (pid file exists and process is alive).
func (s *Ingress) ReloadReady() bool {
	pid, err := s.readPID()
	return err == nil && pid > 0
}

func (s *Ingress) readPID() (int, error) {
	pidFile := s.cfg.Ingress.PidFile
	if !fs.IsExist(pidFile) {
		return 0, fmt.Errorf("pid file(%s) not found", pidFile)
	}
	pidText, err := fs.ReadFileAsString(pidFile)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(pidText))
	if err != nil {
		return 0, err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, err
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, fmt.Errorf("process %d not running: %w", pid, err)
	}
	return pid, nil
}

func (s *Ingress) Reload() error {
	if err := s.ValidateFile(); err != nil {
		return fmt.Errorf("validate before reload: %w", err)
	}
	pidFile := s.cfg.Ingress.PidFile
	pid, err := s.readPID()
	if err != nil {
		return fmt.Errorf("%w — %s", err, ingressReloadHint(s.ConfigPath(), pidFile))
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGHUP)
}
