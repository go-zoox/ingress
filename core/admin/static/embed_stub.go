//go:build !adminui

package static

import (
	"embed"
	"io/fs"
)

//go:embed stub
var ui embed.FS

func uiRoot() (fs.FS, error) {
	return fs.Sub(ui, "stub")
}
