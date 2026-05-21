package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/config"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/ingress/admin/console/app"
	admincfg "github.com/go-zoox/ingress/admin/console/config"
	"github.com/go-zoox/logger"
)

// Admin runs the ingress operations console (HTTP API + optional embedded UI).
func Admin() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "Run the ingress operations admin console",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Admin server config file (YAML)",
				EnvVars: []string{"INGRESS_ADMIN_CONFIG"},
				Value:   "examples/admin-console/admin.yaml",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.String("config")
			if path == "" {
				path = "admin.yaml"
			}

			var cfg admincfg.Config
			if fs.IsExist(path) {
				if err := config.Load(&cfg, &config.LoadOptions{FilePath: path}); err != nil {
					return err
				}
				if err := admincfg.ResolvePaths(&cfg, path); err != nil {
					return err
				}
			} else {
				logger.Warnf("admin config %s not found, using defaults", path)
				if err := admincfg.ResolvePaths(&cfg, ""); err != nil {
					return err
				}
			}

			application, err := app.New(&cfg)
			if err != nil {
				return err
			}
			return application.Run()
		},
	}
}
