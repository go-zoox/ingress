package service

import (
	"os"
	"strings"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/config"
	"github.com/go-zoox/ingress/core/admin/model"
)

// SettingsView is runtime admin configuration and integration status.
type SettingsView struct {
	Admin    AdminSettings    `json:"admin"`
	Ingress  IngressSettings  `json:"ingress"`
	Database DatabaseSettings `json:"database"`
	Logs     LogsSettings     `json:"logs"`
}

type AdminSettings struct {
	Port       int64  `json:"port"`
	DevProxy   bool   `json:"dev_proxy"`
	UIEmbedded bool   `json:"ui_embedded"`
	Enabled    bool   `json:"enabled"`
}

type IngressSettings struct {
	ConfigPath  string `json:"config_path"`
	PidFile     string `json:"pid_file"`
	LogPath     string `json:"log_path"`
	ErrorLogPath string `json:"error_log_path"`
	ReloadReady bool   `json:"reload_ready"`
	ConfigHash  string `json:"config_hash"`
}

type DatabaseSettings struct {
	Driver     string `json:"driver"`
	DSN        string `json:"dsn"`
	WAFEvents  int64  `json:"waf_events"`
	AuditLogs  int64  `json:"audit_logs"`
	Revisions  int64  `json:"config_revisions"`
}

type LogsSettings struct {
	AccessConfigured bool `json:"access_configured"`
	AccessExists     bool `json:"access_exists"`
	ErrorConfigured  bool `json:"error_configured"`
	ErrorExists      bool `json:"error_exists"`
}

// Settings reads admin server configuration for the settings page.
type Settings struct {
	cfg     *config.Config
	ingress *Ingress
	logs    *Logs
}

func NewSettings(cfg *config.Config, ingress *Ingress, logs *Logs) *Settings {
	return &Settings{cfg: cfg, ingress: ingress, logs: logs}
}

func (s *Settings) Get(configHash string) SettingsView {
	out := SettingsView{
		Admin: AdminSettings{
			Port:       s.cfg.Port,
			DevProxy:   s.cfg.Web.DevProxy,
			UIEmbedded: !s.cfg.Web.DevProxy,
			Enabled:    s.cfg.Enabled,
		},
		Ingress: IngressSettings{
			ConfigPath:   s.cfg.IngressConfigPath,
			PidFile:      s.cfg.PidFile,
			LogPath:      s.cfg.LogPath,
			ErrorLogPath: s.cfg.ErrorLogPath,
			ReloadReady:  s.ingress.ReloadReady(),
			ConfigHash:   configHash,
		},
		Database: DatabaseSettings{
			Driver: s.cfg.DatabaseEngine(),
			DSN:    displayDSN(s.cfg.Database.DSN),
		},
		Logs: LogsSettings{
			AccessConfigured: strings.TrimSpace(s.logs.AccessLogPath()) != "",
			AccessExists:     fileExists(s.logs.AccessLogPath()),
			ErrorConfigured:  strings.TrimSpace(s.logs.ErrorLogPath()) != "",
			ErrorExists:      fileExists(s.logs.ErrorLogPath()),
		},
	}
	db := gormx.GetDB()
	if db != nil {
		var n int64
		_ = db.Model(&model.WAFEvent{}).Count(&n).Error
		out.Database.WAFEvents = n
		_ = db.Model(&model.AuditLog{}).Count(&n).Error
		out.Database.AuditLogs = n
		_ = db.Model(&model.ConfigRevision{}).Count(&n).Error
		out.Database.Revisions = n
	}
	return out
}

func displayDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if strings.HasPrefix(dsn, "file:") {
		return dsn
	}
	return dsn
}

func fileExists(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}
