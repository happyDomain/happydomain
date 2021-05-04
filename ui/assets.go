package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:generate yarn --offline build
//go:embed dist

var _assets embed.FS

var Assets http.FileSystem

func init() {
	sub, err := fs.Sub(_assets, "dist")
	if err != nil {
		log.Fatal("Unable to cd to dist/ directory:", err)
	}
	Assets = http.FS(sub)
}
