package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/config"
	zooxfmt "github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/admin/bootstrap"
	"github.com/go-zoox/ingress/core/admin/service"
	"github.com/go-zoox/logger"
)

func MetricsMigrate() *cli.Command {
	return &cli.Command{
		Name:  "metrics-migrate",
		Usage: "Import access.log into admin metrics minute buckets (offline)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Ingress configuration file",
				Aliases: []string{"c"},
				EnvVars: []string{"CONFIG"},
			},
			&cli.StringFlag{
				Name:  "access-log",
				Usage: "Access log file (default: path from ingress logging / admin config)",
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "Only import lines at or after this time (RFC3339 or duration ago, e.g. 24h, 7d)",
			},
			&cli.StringFlag{
				Name:  "until",
				Usage: "Only import lines before this time (RFC3339 or duration ago, e.g. 1h)",
			},
			&cli.BoolFlag{
				Name:  "replace",
				Usage: "Delete existing minute buckets in [since, until) before import (requires both since and until)",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Parse and count without writing to the database",
			},
		},
		Action: func(c *cli.Context) error {
			configFilePath := c.String("config")
			if configFilePath == "" {
				configFilePath = "/etc/ingress/config.yaml"
			}
			if !fs.IsExist(configFilePath) {
				return fmt.Errorf("config file(%s) not found", configFilePath)
			}

			var cfg core.Config
			if err := config.Load(&cfg, &config.LoadOptions{FilePath: configFilePath}); err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if err := core.FinalizeLoadedConfig(&cfg, configFilePath, nil); err != nil {
				return fmt.Errorf("prepare config: %w", err)
			}
			if !cfg.Admin.Enabled {
				return fmt.Errorf("admin.enabled must be true to migrate metrics")
			}

			adminCfg, err := buildAdminConfig(nil, &cfg, configFilePath, "", nil)
			if err != nil {
				return err
			}
			adminCfg.Enabled = true

			accessPath := c.String("access-log")
			if accessPath == "" {
				accessPath = adminCfg.AccessLogPath
			}
			if accessPath == "" {
				return fmt.Errorf("access log path is empty; set logging.transports or --access-log")
			}
			if !fs.IsExist(accessPath) {
				return fmt.Errorf("access log(%s) not found", accessPath)
			}

			opts := service.MigrateAccessLogOptions{DryRun: c.Bool("dry-run"), Replace: c.Bool("replace")}
			if since := c.String("since"); since != "" {
				t, err := parseMigrateTime(since, true)
				if err != nil {
					return fmt.Errorf("--since: %w", err)
				}
				opts.Since = t
			} else {
				opts.Since = time.Now().Add(-rollupPersistRetentionDefault)
			}
			if until := c.String("until"); until != "" {
				t, err := parseMigrateTime(until, true)
				if err != nil {
					return fmt.Errorf("--until: %w", err)
				}
				opts.Until = t
			}
			if opts.Replace && (opts.Since.IsZero() || opts.Until.IsZero()) {
				return fmt.Errorf("--replace requires both --since and --until")
			}

			if err := bootstrap.Init(adminCfg); err != nil {
				return fmt.Errorf("admin db: %w", err)
			}

			logger.Infof("migrating access log %s (since=%s dry_run=%v)", accessPath, opts.Since.Format(time.RFC3339), opts.DryRun)
			res, err := service.MigrateAccessLogToBuckets(accessPath, opts)
			if err != nil {
				return err
			}
			logger.Infof("metrics migrate done: lines_read=%d parsed=%d skipped=%d inserted=%d replaced=%d",
				res.LinesRead, res.LinesParsed, res.LinesSkipped, res.MinutesInserted, res.MinutesReplaced)
			if logger.IsDebugLevel() {
				zooxfmt.PrintJSON("result", res)
			}
			return nil
		},
	}
}

const rollupPersistRetentionDefault = 26 * time.Hour

func parseMigrateTime(raw string, durationAsAgo bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if durationAsAgo {
		if d, ok := parseMigrateDuration(raw); ok {
			return time.Now().Add(-d), nil
		}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected RFC3339 or duration (e.g. 24h, 7d), got %q", raw)
}

// parseMigrateDuration supports Go durations plus day suffix (7d, 1d).
func parseMigrateDuration(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if strings.HasSuffix(raw, "d") || strings.HasSuffix(raw, "D") {
		num := strings.TrimSuffix(strings.TrimSuffix(raw, "d"), "D")
		num = strings.TrimSpace(num)
		if num == "" {
			return 0, false
		}
		var days int
		if _, err := fmt.Sscanf(num, "%d", &days); err != nil || days < 0 {
			return 0, false
		}
		return time.Duration(days) * 24 * time.Hour, true
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return 0, false
	}
	return d, true
}
