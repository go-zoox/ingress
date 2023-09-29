package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/ingress"
	"github.com/go-zoox/ingress/core"
	"github.com/go-zoox/logger"
)

func main() {
	app := cli.NewSingleProgram(&cli.SingleProgramConfig{
		Name:        "ingress",
		Usage:       "Reverse Proxy",
		Description: "An Easy Self Hosted Reverse Proxy",
		Version:     ingress.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "config",
				// Value:   "conf/ingress.yaml",
				Usage:   "The path to the configuration file",
				Aliases: []string{"c"},
				// Required: true,
			},
			&cli.StringFlag{
				Name:    "port",
				Usage:   "The port to listen on",
				Aliases: []string{"p"},
			},
		},
	})

	app.Command(func(c *cli.Context) error {
		configFilePath := c.String("config")
		if configFilePath == "" {
			configFilePath = "/etc/ingress/ingress.yaml"
		}

		var cfg core.Config
		cfg.Port = c.Int64("port")

		if configFilePath != "" {
			if !fs.IsExist(configFilePath) {
				return fmt.Errorf("config file(%s) not found", configFilePath)
			}

			if err := config.Load(&cfg, &config.LoadOptions{
				FilePath: configFilePath,
			}); err != nil {
				return fmt.Errorf("failed to read config file: %s", err)
			}
		}

		if cfg.Port == 0 {
			cfg.Port = 8080
		}

		// @TODO
		if logger.IsDebugLevel() {
			// logger.Debug("config: %v", cfg)
			fmt.PrintJSON("config:", cfg)
		}

		app, err := core.New(ingress.Version, &cfg)
		if err != nil {
			return fmt.Errorf("failed to create core: %s", err)
		}

		return app.Run()
	})

	if err := app.RunWithError(); err != nil {
		logger.Fatal("%s", err.Error())
	}
}
