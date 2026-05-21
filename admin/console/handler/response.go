package handler

import (
	"net/http"

	"github.com/go-zoox/zoox"
)

func ok(ctx *zoox.Context, result any) {
	ctx.JSON(http.StatusOK, zoox.H{
		"code":    200,
		"message": "",
		"result":  result,
	})
}

func fail(ctx *zoox.Context, status int, message string) {
	ctx.JSON(status, zoox.H{
		"code":    status,
		"message": message,
		"result":  nil,
	})
}
