//go:build !prod

package server

import (
	"os"
	"path/filepath"
)

func init() {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		webFS = os.DirFS(filepath.Join(dir, "web", "dist"))
	} else {
		webFS = os.DirFS("web/dist")
	}
}
