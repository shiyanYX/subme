//go:build !prod

package server

import (
	"io/fs"
	"os"
	"path/filepath"
)

func init() {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if f, e := fs.Stat(os.DirFS(filepath.Join(dir, "web", "dist")), "."); e == nil && f.IsDir() {
			webFS = os.DirFS(filepath.Join(dir, "web", "dist"))
			return
		}
	}
	webFS = os.DirFS("web/dist")
}
