//go:build ui

package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:generate npm run build
//go:embed all:build

var _assets embed.FS

var Assets http.FileSystem

func GetEmbedFS() embed.FS {
	return _assets
}

func init() {
	sub, err := fs.Sub(_assets, "build")
	if err != nil {
		log.Fatal("Unable to cd to build/ directory:", err)
	}
	Assets = http.FS(sub)
}
