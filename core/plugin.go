package core

import (
	"net/http"

	"github.com/go-zoox/zoox"
)

type Plugin interface {
	// prepare
	Prepare(app *zoox.Application, cfg *Config) (err error)

	// request
	OnRequest(ctx *zoox.Context, req *http.Request) (err error)

	// response
	OnResponse(ctx *zoox.Context, res *http.Response) (err error)
}

func (c *core) Plugin(plugin ...Plugin) Core {
	c.plugins = append(c.plugins, plugin...)
	return c
}
