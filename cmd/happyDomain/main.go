// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/earthboundkid/versioninfo/v2"
	"github.com/fatih/color"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/app"
	"git.happydns.org/happyDomain/internal/config"
	_ "git.happydns.org/happyDomain/internal/storage/inmemory"
	_ "git.happydns.org/happyDomain/internal/storage/leveldb"
	_ "git.happydns.org/happyDomain/internal/storage/oracle-nosql"
	"git.happydns.org/happyDomain/model"
	_ "git.happydns.org/happyDomain/services/abstract"
	_ "git.happydns.org/happyDomain/services/providers/google"
)

var (
	Version = "custom-build"
)

func main() {
	var err error

	controller.HDVersion = happydns.VersionResponse{
		Version:    Version,
		LastCommit: versioninfo.Revision,
		DirtyBuild: versioninfo.DirtyBuild,
	}
	if Version == "custom-build" {
		controller.HDVersion.Version = versioninfo.Short()
	} else {
		versioninfo.Version = Version
	}

	log.Println("This is happyDomain", versioninfo.Short())
	rand.Seed(time.Now().UTC().UnixNano())

	// Disabled colors in dnscontrol corrections
	color.NoColor = true

	// Load and parse options
	var opts *happydns.Options
	if opts, err = config.ConsolidateConfig(); err != nil {
		log.Fatal(err)
	}

	if opts.NoAuth {
		log.Println("WARNING: NoAuth option must be used for testing or private use behind another restriction/authentication method.")
	}

	// Prepare graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	a := app.NewApp(opts)

	var adminSrv *app.Admin
	if opts.AdminBind != "" {
		adminSrv := app.NewAdmin(a)
		go adminSrv.Start()
	}

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
