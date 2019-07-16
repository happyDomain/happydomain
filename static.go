package main

import (
	"net/http"
	"path"

	"git.nemunai.re/libredns/api"

	"github.com/julienschmidt/httprouter"
)

var StaticDir = "./htdocs/"

func init() {
	api.Router().GET("/", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, "index.html"))
	})
	api.Router().GET("/zones/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, "index.html"))
	})

	api.Router().GET("/css/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, r.URL.Path))
	})
	api.Router().GET("/fonts/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, r.URL.Path))
	})
	api.Router().GET("/img/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, r.URL.Path))
	})
	api.Router().GET("/js/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, r.URL.Path))
	})
	api.Router().GET("/views/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.ServeFile(w, r, path.Join(StaticDir, r.URL.Path))
	})
}
