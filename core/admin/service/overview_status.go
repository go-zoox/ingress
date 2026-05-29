package service

import (
	"time"

	ingpkg "github.com/go-zoox/ingress"
	ingcore "github.com/go-zoox/ingress/core"
)

// OverviewStatus is ingress runtime metadata for the admin overview page.
type OverviewStatus struct {
	Version             string `json:"version"`
	ConfigPath          string `json:"config_path"`
	PidFile             string `json:"pid_file,omitempty"`
	ReloadReady         bool   `json:"reload_ready"`
	ListenHTTP          int64  `json:"listen_http"`
	ListenHTTPS         int64  `json:"listen_https"`
	RulesCount          int    `json:"rules_count"`
	WAFEnabled          bool   `json:"waf_enabled"`
	WAFLogOnly          bool   `json:"waf_log_only"`
	WAFRuntimeEnabled   bool   `json:"waf_runtime_enabled"`
	LastReload          string `json:"last_reload"`
	ConfigHash          string `json:"config_hash"`
	FileHash            string `json:"file_hash"`
	RuntimeHash         string `json:"runtime_hash"`
	LatestRevisionHash  string `json:"latest_revision_hash"`
	RuntimeDrift        bool   `json:"runtime_drift"`
	RevisionDrift       bool   `json:"revision_drift"`
}

// BuildOverviewStatus assembles the status block for overview snapshots.
func BuildOverviewStatus(ingress *Ingress, config *Config) OverviewStatus {
	out := OverviewStatus{
		Version:    ingpkg.Version,
		LastReload: time.Now().Format(time.RFC3339),
	}
	if ingress == nil {
		return out
	}

	out.ConfigPath = ingress.ConfigPath()
	if ingress.cfg != nil {
		out.PidFile = ingress.cfg.PidFile
		if ingress.cfg.CoreInstance != nil {
			out.WAFRuntimeEnabled = ingress.cfg.CoreInstance.IsWAFEnabled()
		}
	}

	icfg, err := ingress.LoadConfig()
	if err == nil && icfg != nil {
		out.WAFEnabled = icfg.WAF.Enabled
		out.WAFLogOnly = icfg.WAF.LogOnly
		out.ListenHTTP = icfg.Port
		out.ListenHTTPS = icfg.HTTPS.Port
		out.RulesCount = len(icfg.Rules)
		if !out.WAFRuntimeEnabled {
			out.WAFRuntimeEnabled = icfg.WAF.Enabled
		}
	}

	out.ReloadReady = ingress.ReloadReady()
	fileHash := overviewFileConfigHash(ingress)
	runtimeHash := overviewRuntimeConfigHash(ingress)
	out.ConfigHash = fileHash
	out.FileHash = fileHash
	out.RuntimeHash = runtimeHash

	if config != nil {
		if revs, err := config.ListRevisions(1); err == nil && len(revs) > 0 {
			out.LatestRevisionHash = revs[0].Hash
		}
	}
	out.RuntimeDrift = runtimeHash != "" && fileHash != "" && runtimeHash != fileHash
	out.RevisionDrift = out.LatestRevisionHash != "" && fileHash != "" && fileHash != out.LatestRevisionHash

	return out
}

func overviewFileConfigHash(ingress *Ingress) string {
	if ingress == nil {
		return ""
	}
	content, err := ingress.ReadYAML()
	if err != nil {
		return ""
	}
	return ingcore.ContentHash(content)
}

func overviewRuntimeConfigHash(ingress *Ingress) string {
	if ingress == nil || ingress.cfg == nil || ingress.cfg.CoreInstance == nil {
		return ""
	}
	return ingress.cfg.CoreInstance.ConfigFingerprint()
}
