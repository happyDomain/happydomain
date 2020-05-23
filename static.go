// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"

	"github.com/julienschmidt/httprouter"
)

//go:generate yarn --cwd htdocs --offline build
//go:generate go-bindata -ignore "\\.go|\\.less" -pkg "main" -o "bindata.go" htdocs/dist/...
//go:generate go fmt bindata.go
//go:generate go-bindata -ignore "\\.go|\\.less" -pkg "utils" -o "utils/bindata.go" htdocs/dist/img/happydns.png
//go:generate go fmt utils/bindata.go

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

	// Create a dedicated route for all assets not behind a known static directory
	rootFiles, _ := AssetDir(StaticDir)
	for _, rfile := range rootFiles {
		if _, err := AssetDir(path.Join(StaticDir, rfile)); err != nil {
			api.Router().GET("/"+rfile, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				opts := r.Context().Value("opts").(*config.Options)

				if opts.DevProxy == "" {
					if data, err := Asset(path.Join(StaticDir, r.URL.Path)); err == nil {
						if strings.HasSuffix(r.URL.Path, ".js") {
							w.Header().Set("Content-Type", "text/javascript")
						} else if strings.HasSuffix(r.URL.Path, ".json") {
							w.Header().Set("Content-Type", "application/json")
						} else if strings.HasSuffix(r.URL.Path, ".css") {
							w.Header().Set("Content-Type", "text/css")
						}
						w.Write(data)
					} else {
						fmt.Fprintf(w, "{\"errmsg\":%q}", err)
					}
				} else {
					fwd_request(w, r, opts.DevProxy)
				}
			})
		}
	}

	api.Router().GET("/domains/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	api.Router().GET("/en/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	api.Router().GET("/fr/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	api.Router().GET("/email-validation", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	api.Router().GET("/sources/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	api.Router().GET("/tools/*_", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
