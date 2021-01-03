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
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"git.happydns.org/happydns/admin"
	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"

	_ "git.happydns.org/happydns/sources/alwaysdata"
	_ "git.happydns.org/happydns/sources/ddns"
	_ "git.happydns.org/happydns/sources/gandi"
	_ "git.happydns.org/happydns/sources/ovh"

	_ "git.happydns.org/happydns/services/providers/google"

	_ "git.happydns.org/happydns/storage/leveldb"
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

	// Initialize storage
	if s, ok := storage.StorageEngines[opts.StorageEngine]; !ok {
		log.Fatal(fmt.Sprintf("Unexistant storage engine: %q, please select one between: %v", opts.StorageEngine, storage.GetStorageEngines()))
	} else {
		log.Println("Opening database...")
		if store, err := s(); err != nil {
			log.Fatal("Cannot open the database: ", err)
		} else {
			defer store.Close()
			storage.MainStore = store
		}
	}

	if opts.NoAuth {
		// Check if the default account exists.
		if !storage.MainStore.UserExists(api.NO_AUTH_ACCOUNT) {
			if user, err := happydns.NewUser(api.NO_AUTH_ACCOUNT, ""); err != nil {
				log.Fatal("Unable to create default account:", err)
			} else {
				user.Settings = *happydns.DefaultUserSettings()
				if err := storage.MainStore.CreateUser(user); err != nil {
					log.Fatal("Unable to create default account in database:", err)
				} else {
					log.Println("Default account for NoAuth created.")
				}
			}
		}
		log.Println("WARNING: NoAuth option has to be use for testing or personnal purpose behind another restriction/authentication method.")
	}

	log.Println("Do database migrations...")
	if err = storage.MainStore.DoMigration(); err != nil {
		log.Fatal("Cannot migrate database: ", err)
	}

	// Prepare graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	if opts.AdminBind != "" {
		adminSrv := &http.Server{
			Addr:    opts.AdminBind,
			Handler: StripPrefix(opts, admin.Router()),
		}

		go func() {
			if !strings.Contains(opts.AdminBind, ":") {
				if _, err := os.Stat(opts.AdminBind); !os.IsNotExist(err) {
					if err := os.Remove(opts.AdminBind); err != nil {
						log.Fatal(err)
					}
				}

				unixListener, err := net.Listen("unix", opts.AdminBind)
				if err != nil {
					log.Fatal(err)
				}
				log.Fatal(adminSrv.Serve(unixListener))
			} else {
				log.Fatal(adminSrv.ListenAndServe())
			}
		}()
		log.Println(fmt.Sprintf("Admin listening on %s", opts.AdminBind))
	}

	srv := &http.Server{
		Addr:    opts.Bind,
		Handler: StripPrefix(opts, api.Router()),
	}

	// Serve content
	go func() {
		log.Fatal(srv.ListenAndServe())
	}()
	log.Println(fmt.Sprintf("Ready, listening on %s", opts.Bind))

	// Wait shutdown signal
	<-interrupt

	log.Print("The service is shutting down...")
	srv.Shutdown(context.Background())
	log.Println("done")
}
