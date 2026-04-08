package main

import (
	"os"

	"github.com/go-zoox/cli"
	"github.com/go-zoox/config"
	"github.com/go-zoox/core-utils/fmt"
	"github.com/go-zoox/fs"
	"github.com/go-zoox/ingress/core"
	"github.com/go-zoox/logger"
	"gopkg.in/yaml.v3"
)

func Validate() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate ingress configuration file syntax",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "The path to the configuration file",
				Aliases: []string{"c"},
				EnvVars: []string{"CONFIG"},
			},
		},
		Action: func(c *cli.Context) error {
			configFilePath := c.String("config")
			if configFilePath == "" {
				configFilePath = "/etc/ingress/ingress.yaml"
			}

			if err := validateConfigFile(configFilePath); err != nil {
				return err
			}

			logger.Infof("config file(%s) is valid", configFilePath)
			return nil
		},
	}
}

func validateConfigFile(configFilePath string) error {
	if !fs.IsExist(configFilePath) {
		return fmt.Errorf("config file(%s) not found", configFilePath)
	}

	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read config file(%s): %s", configFilePath, err)
	}

	var yamlNode map[string]interface{}
	if err := yaml.Unmarshal(content, &yamlNode); err != nil {
		return fmt.Errorf("yaml syntax error in config file(%s): %s", configFilePath, err)
	}

	var cfg core.Config
	if err := config.Load(&cfg, &config.LoadOptions{
		FilePath: configFilePath,
	}); err != nil {
		return fmt.Errorf("invalid config format in file(%s): %s", configFilePath, err)
	}

	if err := core.ValidateConfig(&cfg); err != nil {
		return fmt.Errorf("unsupported configuration in file(%s): %s", configFilePath, err)
	}

	return nil
}
