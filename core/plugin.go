package core

import (
	"net/http"

	"github.com/go-zoox/zoox"
)

type Plugin interface {
	OnRequest(ctx *zoox.Context, req *http.Request) (err error)
	OnResponse(ctx *zoox.Context, res http.ResponseWriter) (err error)
}
