package main

import (
	"github.com/go-zoox/cli"
	"github.com/go-zoox/ingress"
	"github.com/go-zoox/logger"
)

var pidFilePath string = "/tmp/gozoox.ingress.pid"

func main() {
	app := cli.NewMultipleProgram(&cli.MultipleProgramConfig{
		Name:        "ingress",
		Usage:       "Reverse Proxy",
		Description: "An Easy Self Hosted Reverse Proxy",
		Version:     ingress.Version,
	})

	app.Register("run", Run())
	app.Register("reload", Reload())

	if err := app.RunWithError(); err != nil {
		logger.Fatal("%s", err.Error())
	}
}
