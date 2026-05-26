//go:build adminui

package static

import (
	"embed"
	"io/fs"
)

//go:embed dist
var ui embed.FS

func uiRoot() (fs.FS, error) {
	return fs.Sub(ui, "dist")
}
