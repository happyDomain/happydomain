package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"

	"github.com/julienschmidt/httprouter"
)

//go:generate yarn --cwd htdocs --offline build
//go:generate go-bindata -ignore "\\.go|\\.less" -pkg "main" -o "bindata.go" htdocs/dist/...
//go:generate go fmt bindata.go

const StaticDir string = "htdocs/dist"

func init() {
	api.Router().GET("/", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset("htdocs/dist/index.html"); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/join", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset("htdocs/dist/index.html"); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			r.URL.Path = "/"
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/login", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset("htdocs/dist/index.html"); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			r.URL.Path = "/"
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/services/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset("htdocs/dist/index.html"); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			r.URL.Path = "/"
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/zones/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset("htdocs/dist/index.html"); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			r.URL.Path = "/"
			fwd_request(w, r, opts.DevProxy)
		}
	})

	api.Router().GET("/css/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Header().Set("Content-Type", "text/css")
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/fonts/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/img/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/js/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Header().Set("Content-Type", "text/javascript")
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})

	api.Router().GET("/favicon.ico", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/manifest.json", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/robots.txt", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
	api.Router().GET("/service-worker.js", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		opts := r.Context().Value("opts").(*config.Options)

		if opts.DevProxy == "" {
			if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err != nil {
				fmt.Fprintf(w, "{\"errmsg\":%q}", err)
			} else {
				w.Header().Set("Content-Type", "text/javascript")
				w.Write(data)
			}
		} else {
			fwd_request(w, r, opts.DevProxy)
		}
	})
}

func fwd_request(w http.ResponseWriter, r *http.Request, fwd string) {
	if u, err := url.Parse(fwd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		u.Path = path.Join(u.Path, r.URL.Path)

		if r, err := http.NewRequest(r.Method, u.String(), r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else if resp, err := http.DefaultClient.Do(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
		} else {
			defer resp.Body.Close()

			for key := range resp.Header {
				w.Header().Add(key, resp.Header.Get(key))
			}
			w.WriteHeader(resp.StatusCode)

			io.Copy(w, resp.Body)
		}
	}
}
