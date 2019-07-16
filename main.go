package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"

	"git.nemunai.re/libredns/api"
)

var DefaultNameServer = "127.0.0.1:53"

type ResponseWriterPrefix struct {
	real   http.ResponseWriter
	prefix string
}

func (r ResponseWriterPrefix) Header() http.Header {
	return r.real.Header()
}

func (r ResponseWriterPrefix) WriteHeader(s int) {
	if v, exists := r.real.Header()["Location"]; exists {
		r.real.Header().Set("Location", r.prefix+v[0])
	}
	r.real.WriteHeader(s)
}

func (r ResponseWriterPrefix) Write(z []byte) (int, error) {
	return r.real.Write(z)
}

func StripPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if prefix != "/" && r.URL.Path == "/" {
			http.Redirect(w, r, prefix+"/", http.StatusFound)
		} else if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			h.ServeHTTP(ResponseWriterPrefix{w, prefix}, r2)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func main() {
	// Read parameters from command line
	var bind = flag.String("bind", ":8081", "Bind port/socket")
	var dsn = flag.String("dsn", DSNGenerator(), "DSN to connect to the MySQL server")
	var baseURL = flag.String("baseurl", "/", "URL prepended to each URL")
	flag.StringVar(&DefaultNameServer, "defaultns", DefaultNameServer, "Adress to the default name server")
	flag.Parse()

	// Sanitize options
	if *baseURL != "/" {
		tmp := path.Clean(*baseURL)
		baseURL = &tmp
	} else {
		tmp := ""
		baseURL = &tmp
	}

	// Initialize contents
	log.Println("Opening database...")
	if err := DBInit(*dsn); err != nil {
		log.Fatal("Cannot open the database: ", err)
	}
	defer DBClose()

	log.Println("Creating database...")
	if err := DBCreate(); err != nil {
		log.Fatal("Cannot create database: ", err)
	}

	// Serve content
	log.Println("Ready, listening on", *bind)
	if err := http.ListenAndServe(*bind, StripPrefix(*baseURL, api.Router())); err != nil {
		log.Fatal("Unable to listen and serve: ", err)
	}
}
