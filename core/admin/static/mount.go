package static

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-zoox/zoox"
)

func Mount(app *zoox.Application) error {
	sub, err := uiRoot()
	if err != nil {
		return mountAPIOnly(app)
	}

	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return mountAPIOnly(app)
	}

	shell := indexHTML
	fsys := http.FS(sub)

	app.Use(func(ctx *zoox.Context) {
		if ctx.Method != http.MethodGet && ctx.Method != http.MethodHead {
			ctx.Next()
			return
		}
		p := ctx.Request.URL.Path
		if p == "/" {
			ctx.Data(http.StatusOK, "text/html; charset=utf-8", shell)
			return
		}
		if strings.HasPrefix(p, "/api/") {
			ctx.Next()
			return
		}
		if strings.Contains(p, ".") {
			ctx.Next()
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", shell)
	})
	app.StaticFS("/", fsys)
	return nil
}

func mountAPIOnly(app *zoox.Application) error {
	app.Get("/", func(ctx *zoox.Context) {
		ctx.String(http.StatusOK, "ingress admin API — build web: make -C core/admin web && go build -tags adminui ./cmd/ingress")
	})
	return nil
}
