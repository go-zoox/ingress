package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/ingress"
	"github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/waf"
	"github.com/go-zoox/logger"
)

func Run() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run the ingress server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "config",
				// Example dev config: ingress run -c examples/basic/ingress.yaml
				Usage:   "The path to the configuration file",
				Aliases: []string{"c"},
				EnvVars: []string{"CONFIG"},
			},
			&cli.StringFlag{
				Name:    "port",
				Usage:   "The port to listen on",
				Aliases: []string{"p"},
				EnvVars: []string{"PORT"},
			},
			&cli.StringFlag{
				Name:  "pid-file",
				Usage: "The path to the pid file",
				Value: "/tmp/gozoox.ingress.pid",
			},
		},
		Action: func(c *cli.Context) error {
			configFilePath := c.String("config")
			if configFilePath == "" {
				configFilePath = "/etc/ingress/config.yaml"
			}

			// @TODO
			if c.String("pid-file") != "" {
				pidFilePath = c.String("pid-file")
			}

			var cfg core.Config

			if configFilePath != "" {
				if !fs.IsExist(configFilePath) {
					return fmt.Errorf("config file(%s) not found", configFilePath)
				}

				if err := config.Load(&cfg, &config.LoadOptions{
					FilePath: configFilePath,
				}); err != nil {
					return fmt.Errorf("failed to read config file: %s", err)
				}
				if err := core.ResolveConfigPaths(&cfg, configFilePath); err != nil {
					return fmt.Errorf("failed to resolve config paths: %w", err)
				}
				if err := waf.ApplyRulePatchesFromFile(configFilePath, cfg.Rules); err != nil {
					return fmt.Errorf("failed to apply rules[].waf patches: %w", err)
				}
			}

			if c.Int64("port") != 0 {
				cfg.Port = c.Int64("port")
			}

			if cfg.Port == 0 {
				cfg.Port = 8080
			}

			// @TODO
			if logger.IsDebugLevel() {
				// logger.Debug("config: %v", cfg)
				fmt.PrintJSON("config:", cfg)
			}

			app, err := core.NewWithPaths(ingress.Version, &cfg, configFilePath, pidFilePath)
			if err != nil {
				return fmt.Errorf("failed to create core: %s", err)
			}

			if cfg.Admin.Enabled {
				if err := startAdmin(app, &cfg, configFilePath, pidFilePath); err != nil {
					return err
				}
			}

			signals := make(chan os.Signal, 1)
			signal.Notify(signals, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

			go func() {
				for {
					sig := <-signals
					switch sig {
					case syscall.SIGHUP:
						var cfg core.Config
						if err := config.Load(&cfg, &config.LoadOptions{
							FilePath: configFilePath,
						}); err != nil {
							logger.Errorf("failed to read config file: %s", err)
							return
						}
						if err := core.ResolveConfigPaths(&cfg, configFilePath); err != nil {
							logger.Errorf("failed to resolve config paths: %s", err)
							return
						}
						if err := waf.ApplyRulePatchesFromFile(configFilePath, cfg.Rules); err != nil {
							logger.Errorf("failed to apply rules[].waf patches: %s", err)
							return
						}

						// @TODO
						if logger.IsDebugLevel() {
							// logger.Debug("config: %v", cfg)
							fmt.PrintJSON("config:", cfg)
						}

						if err := app.Reload(&cfg); err != nil {
							logger.Errorf("failed to reload: %s", err)
						}

						logger.Infof("server reload")

					case syscall.SIGTERM, syscall.SIGINT:
						os.Exit(0)
					default:
						logger.Warn("unknown signal: %s", sig)
					}
				}
			}()

			fs.WriteFile(pidFilePath, []byte(fmt.Sprintf("%d", os.Getpid())))

			return app.Run()
		},
	}
}
