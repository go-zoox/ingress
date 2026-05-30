package service

import (
	"fmt"

	zcfg "github.com/go-zoox/config"
	ingcore "github.com/go-zoox/ingress/core"
)

// Scenarios wraps scenario list/active operations for Admin Console.
type Scenarios struct {
	ingress *Ingress
	audit   *Audit
	config  *Config
}

func NewScenarios(ingress *Ingress, audit *Audit, config *Config) *Scenarios {
	return &Scenarios{ingress: ingress, audit: audit, config: config}
}

// List returns scenario metadata from the on-disk config (env override not reflected in YAML).
func (s *Scenarios) List() (ingcore.ScenariosResponse, error) {
	cfg, err := s.loadRawConfig()
	if err != nil {
		return ingcore.ScenariosResponse{}, err
	}
	if !cfg.Scenarios.Configured() {
		return ingcore.ScenariosResponse{}, nil
	}
	return ingcore.ListScenarios(cfg), nil
}

// SetActive updates scenarios.active in ingress.yaml, validates, saves, and reloads ingress.
func (s *Scenarios) SetActive(id string) (ingcore.ScenariosResponse, error) {
	content, err := s.ingress.ReadYAML()
	if err != nil {
		return ingcore.ScenariosResponse{}, err
	}
	updated, err := ingcore.SetScenariosActiveYAML(content, id)
	if err != nil {
		return ingcore.ScenariosResponse{}, err
	}
	if err := s.ingress.ValidateYAML(updated); err != nil {
		return ingcore.ScenariosResponse{}, fmt.Errorf("validate after scenario switch: %w", err)
	}
	if _, err := s.config.Save(updated, "scenario:"+id); err != nil {
		return ingcore.ScenariosResponse{}, err
	}
	if err := s.ingress.Reload(); err != nil {
		return ingcore.ScenariosResponse{}, err
	}
	_ = s.audit.Record("scenario.activate", id, "admin")
	cfg, err := s.loadRawConfig()
	if err != nil {
		return ingcore.ScenariosResponse{}, err
	}
	return ingcore.ListScenarios(cfg), nil
}

func (s *Scenarios) loadRawConfig() (*ingcore.Config, error) {
	var cfg ingcore.Config
	if err := zcfg.Load(&cfg, &zcfg.LoadOptions{FilePath: s.ingress.ConfigPath()}); err != nil {
		return nil, err
	}
	return &cfg, nil
}
