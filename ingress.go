package main

import (
	"fmt"
	"os"

	"github.com/go-zoox/config"
	"github.com/go-zoox/fs"
	ingress "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/logger"
	"github.com/urfave/cli/v2"

	corePlugin "github.com/go-zoox/ingress/plugins/core"
)

func main() {
	app := &cli.App{
		Name:        "ingress",
		Usage:       "Reverse Proxy",
		Description: "An Easy Self Hosted Reverse Proxy",
		Version:     fmt.Sprintf("%s (%s %s)", Version, CommitHash, BuildTime),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "config",
				// Value:   "conf/ingress.yaml",
				Usage:   "The path to the configuration file",
				Aliases: []string{"c"},
			},
			&cli.StringFlag{
				Name:    "port",
				Usage:   "The port to listen on",
				Aliases: []string{"p"},
			},
		},
		Action: func(c *cli.Context) error {
			configFilePath := c.String("config")
			port := c.Int64("port")
			if configFilePath == "" {
				logger.Error("config file is required with -c or --config")
				os.Exit(1)
			}

			var cfg ingress.Config

			if configFilePath != "" {
				if !fs.IsExist(configFilePath) {
					logger.Error("config file(%s) not found", configFilePath)
					os.Exit(1)
				}

				if err := config.Load(&cfg, &config.LoadOptions{
					FilePath: configFilePath,
				}); err != nil {
					logger.Error("failed to read config file", err)
					os.Exit(1)
				}

				// j, _ := json.MarshalIndent(cfg, "", "  ")
				// fmt.Println(string(j))
				// os.Exit(0)
			}

			if port != 0 {
				cfg.Port = port
			}

			// @TODO
			if os.Getenv("DEBUG") == "true" {
				logger.Debug("config: %v", cfg)
			}

			app := ingress.New(Version, &cfg)

			app.Use(&corePlugin.Core{
				Application: app,
			})

			app.Start()

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("%s", err.Error())
	}
}
