package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
	ingcore "github.com/go-zoox/ingress/core"
)

// ConfigRevisionSummary is a list row for version history.
type ConfigRevisionSummary struct {
	ID        uint      `json:"id"`
	Hash      string    `json:"hash"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// ConfigRevisionDetail includes full YAML snapshot.
type ConfigRevisionDetail struct {
	ConfigRevisionSummary
	Content string `json:"content"`
}

// ConfigRouteImpact describes one routing row change between published and draft configs.
type ConfigRouteImpact struct {
	Kind       string   `json:"kind"` // added | removed | changed
	Host       string   `json:"host"`
	Path       string   `json:"path"`
	RuleIndex  int      `json:"rule_index"`
	PathIndex  int      `json:"path_index"`
	Fields     []string `json:"fields,omitempty"`
	Before     string   `json:"before,omitempty"`
	After      string   `json:"after,omitempty"`
}

// ConfigPreview summarizes draft changes before publish.
type ConfigPreview struct {
	Valid          bool     `json:"valid"`
	Hash           string   `json:"hash"`
	PublishedHash  string   `json:"published_hash"`
	Changed        bool     `json:"changed"`
	Error          string   `json:"error,omitempty"`
	ModulesChanged []string `json:"modules_changed"`
	GlobalTouches  []string `json:"global_touches"`
	RouteImpacts   []ConfigRouteImpact `json:"route_impacts"`
}

// Config coordinates modular editing, preview, and revision history.
type Config struct {
	ingress *Ingress
	audit   *Audit
}

func NewConfig(ingress *Ingress, audit *Audit) *Config {
	return &Config{ingress: ingress, audit: audit}
}

func (c *Config) Modules(content string) ([]ConfigModule, error) {
	if strings.TrimSpace(content) == "" {
		var err error
		content, err = c.ingress.ReadYAML()
		if err != nil {
			return nil, err
		}
	}
	return SplitConfigModules(content)
}

func (c *Config) ApplyModule(content, moduleID, moduleYAML string) (string, error) {
	if strings.TrimSpace(content) == "" {
		var err error
		content, err = c.ingress.ReadYAML()
		if err != nil {
			return "", err
		}
	}
	return MergeConfigModule(content, moduleID, moduleYAML)
}

func (c *Config) Preview(draft string) (*ConfigPreview, error) {
	published, err := c.ingress.ReadYAML()
	if err != nil {
		return nil, err
	}
	out := &ConfigPreview{
		Hash:          configHash(draft),
		PublishedHash: configHash(published),
		Changed:       normalizeYAML(draft) != normalizeYAML(published),
	}
	if err := c.ingress.ValidateYAML(draft); err != nil {
		out.Valid = false
		out.Error = err.Error()
		return out, nil
	}
	out.Valid = true
	changed, err := ChangedConfigModules(published, draft)
	if err != nil {
		return nil, err
	}
	out.ModulesChanged = changed
	out.GlobalTouches = globalTouchesFromModules(changed)
	if out.Valid {
		impacts, err := AnalyzeRouteImpacts(c.ingress, published, draft)
		if err != nil {
			return nil, err
		}
		out.RouteImpacts = impacts
	}
	return out, nil
}

func (c *Config) ListRevisions(limit int) ([]ConfigRevisionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []model.ConfigRevision
	if err := gormx.GetDB().Order("created_at desc, id desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ConfigRevisionSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, ConfigRevisionSummary{
			ID:        row.ID,
			Hash:      row.Hash,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (c *Config) GetRevision(id uint) (*ConfigRevisionDetail, error) {
	var row model.ConfigRevision
	if err := gormx.GetDB().First(&row, id).Error; err != nil {
		return nil, fmt.Errorf("revision not found")
	}
	return &ConfigRevisionDetail{
		ConfigRevisionSummary: ConfigRevisionSummary{
			ID:        row.ID,
			Hash:      row.Hash,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
		},
		Content: row.Content,
	}, nil
}

func (c *Config) Save(content, note string) (string, error) {
	if err := c.ingress.ValidateYAML(content); err != nil {
		return "", err
	}
	if err := c.ingress.WriteYAML(content); err != nil {
		return "", err
	}
	hash := c.recordRevision(content, note)
	_ = c.audit.Record("config.save", hash, "admin")
	return hash, nil
}

func (c *Config) Publish(content, note string) (string, error) {
	if strings.TrimSpace(note) == "" {
		note = "publish"
	}
	hash, err := c.Save(content, note)
	if err != nil {
		return "", err
	}
	if err := c.ingress.Reload(); err != nil {
		return hash, err
	}
	SyncGeoIPFromIngress(c.ingress)
	_ = c.audit.Record("ingress.reload", c.ingress.ConfigPath(), "admin")
	return hash, nil
}

func (c *Config) recordRevision(content, note string) string {
	if strings.TrimSpace(note) == "" {
		note = "save"
	}
	hash := configHash(content)
	_ = gormx.GetDB().Create(&model.ConfigRevision{
		Hash:      hash,
		Content:   content,
		Note:      note,
		CreatedAt: time.Now(),
	}).Error
	return hash
}

func configHash(content string) string {
	return ingcore.ContentHash(content)
}

func normalizeYAML(content string) string {
	modules, err := SplitConfigModules(content)
	if err != nil {
		return strings.TrimSpace(content)
	}
	cur := ""
	for _, mod := range modules {
		if strings.TrimSpace(mod.YAML) == "" {
			continue
		}
		cur, err = MergeConfigModule(cur, mod.ID, mod.YAML)
		if err != nil {
			return strings.TrimSpace(content)
		}
	}
	return strings.TrimSpace(cur)
}
