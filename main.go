// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/internal/app"
	"git.happydns.org/happydomain/storage"

	_ "git.happydns.org/happydomain/services/providers/google"

	_ "git.happydns.org/happydomain/storage/leveldb"
)

var (
	Version = "custom-build"
)

func main() {
	var err error

	log.Println("This is happyDomain", Version)
	rand.Seed(time.Now().UTC().UnixNano())

	// Load and parse options
	var opts *config.Options
	if opts, err = config.ConsolidateConfig(); err != nil {
		log.Fatal(err)
	}

	// Initialize storage
	if s, ok := storage.StorageEngines[opts.StorageEngine]; !ok {
		log.Fatal(fmt.Sprintf("Nonexistent storage engine: %q, please select one of: %v", opts.StorageEngine, storage.GetStorageEngines()))
	} else {
		log.Println("Opening database...")
		if store, err := s(); err != nil {
			log.Fatal("Could not open the database: ", err)
		} else {
			defer store.Close()
			storage.MainStore = store
		}
	}

	if opts.NoAuth {
		log.Println("WARNING: NoAuth option must be used for testing or private use behind another restriction/authentication method.")
	}

	log.Println("Performing database migrations...")
	if err = storage.MainStore.DoMigration(); err != nil {
		log.Fatal("Could not migrate database: ", err)
	}

	// Prepare graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	var adminSrv *app.Admin
	if opts.AdminBind != "" {
		adminSrv := app.NewAdmin(opts)
		go adminSrv.Start()
	}

	a := app.NewApp(opts)
	go a.Start()

	// Wait shutdown signal
	<-interrupt

	log.Println("Stopping the service...")
	a.Stop()
	if adminSrv != nil {
		adminSrv.Stop()
	}
	log.Println("Stopped")
}
