package main

import (
	"os"
	"strconv"
	"syscall"

	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/logger"

	"github.com/go-zoox/cli"
)

func Reload() *cli.Command {
	return &cli.Command{
		Name:  "reload",
		Usage: "Reload the ingress server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "The path to the configuration file",
				Aliases: []string{"c"},
				EnvVars: []string{"CONFIG"},
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
				configFilePath = "/etc/ingress/ingress.yaml"
			}

			if err := validateConfigFile(configFilePath); err != nil {
				return fmt.Errorf("failed to validate config before reload: %s", err)
			}

			// @TODO
			if c.String("pid-file") != "" {
				pidFilePath = c.String("pid-file")
			}

			if !fs.IsExist(pidFilePath) {
				return fmt.Errorf("pid file(%s) not found", pidFilePath)
			}

			pidText, err := fs.ReadFileAsString(pidFilePath)
			if err != nil {
				return err
			}

			pid, err := strconv.Atoi(pidText)
			if err != nil {
				return err
			}

			process, err := os.FindProcess(pid)
			if err != nil {
				return err
			}

			if err := process.Signal(syscall.SIGHUP); err != nil {
				return err
			}

			logger.Infof("reload ingress server success")

			return nil
		},
	}
}
