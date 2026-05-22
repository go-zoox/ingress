package static

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-zoox/zoox"
)

//go:embed dist
var dist embed.FS

// Mount serves the built React app at / when dist is present.
func Mount(app *zoox.Application) error {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// dist not built yet — API-only mode
		app.Get("/", func(ctx *zoox.Context) {
			ctx.String(http.StatusOK, "ingress admin API — build web: cd core/admin/web && pnpm build")
		})
		return nil
	}

	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return err
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
