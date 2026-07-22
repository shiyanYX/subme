//go:build prod

package server

import (
	"embed"
	"io/fs"
)

//go:embed all:web
var embedFS embed.FS

func init() {
	webFS, _ = fs.Sub(embedFS, "web")
}
