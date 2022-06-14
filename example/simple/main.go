package main

import (
	"os"

	"github.com/go-zoox/config"
	ingress "github.com/go-zoox/ingress/core"
	corePlugin "github.com/go-zoox/ingress/plugins/core"
	"github.com/go-zoox/logger"
)

func main() {
	var cfg ingress.Config
	if err := config.Load(&cfg, &config.LoadOptions{
		FilePath: "conf/ingress.yaml",
	}); err != nil {
		logger.Error("failed to read config file", err)
		os.Exit(1)
	}

	app := ingress.New("0.0.0", &cfg)

	app.Use(&corePlugin.Core{
		Application: app,
	})

	app.Start()

}
