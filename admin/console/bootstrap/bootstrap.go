package bootstrap

import (
	"fmt"
	"strings"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/admin/console/config"
	"github.com/go-zoox/ingress/admin/console/model"
)

// Init connects SQLite (or other engines) and migrates admin tables.
func Init(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("bootstrap: config is nil")
	}
	engine := cfg.DatabaseEngine()
	dsn := strings.TrimSpace(cfg.Database.DSN)
	if dsn == "" {
		return fmt.Errorf("bootstrap: database.dsn is required")
	}
	if err := gormx.LoadDB(engine, dsn); err != nil {
		return fmt.Errorf("bootstrap: load db: %w", err)
	}
	db := gormx.GetDB()
	if err := db.AutoMigrate(model.MigrateModels()...); err != nil {
		return fmt.Errorf("bootstrap: automigrate: %w", err)
	}
	if err := seedSampleDataIfEmpty(); err != nil {
		return fmt.Errorf("bootstrap: sample data: %w", err)
	}
	return nil
}
