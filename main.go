package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/storage"
	leveldb "git.happydns.org/happydns/storage/leveldb"
)

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

func StripPrefix(opts *config.Options, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add in the context's request options
		ctx := r.Context()
		ctx = context.WithValue(ctx, "opts", opts)
		r = r.WithContext(ctx)

		if opts.BaseURL != "" && r.URL.Path == "/" {
			http.Redirect(w, r, opts.BaseURL+"/", http.StatusFound)
		} else if p := strings.TrimPrefix(r.URL.Path, opts.BaseURL); len(p) < len(r.URL.Path) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			h.ServeHTTP(ResponseWriterPrefix{w, opts.BaseURL}, r2)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func main() {
	var err error

	rand.Seed(time.Now().UTC().UnixNano())

	// Load and parse options
	var opts *config.Options
	if opts, err = config.ConsolidateConfig(); err != nil {
		log.Fatal(err)
	}

	// Initialize contents
	log.Println("Opening database...")
	if store, err := leveldb.NewLevelDBStorage("happydns.db"); err != nil {
		log.Fatal("Cannot open the database: ", err)
	} else {
		defer store.Close()
		storage.MainStore = store
	}

	log.Println("Do database migrations...")
	if err = storage.MainStore.DoMigration(); err != nil {
		log.Fatal("Cannot migrate database: ", err)
	}

	// Serve content
	log.Println("Ready, listening on", opts.Bind)
	if err = http.ListenAndServe(opts.Bind, StripPrefix(opts, api.Router())); err != nil {
		log.Fatal("Unable to listen and serve: ", err)
	}
}
