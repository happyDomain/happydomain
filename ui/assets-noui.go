//go:build !ui

package ui

import (
	"net/http"
)

var Assets http.FileSystem

func GetEmbedFS() http.FileSystem {
	return Assets
}
