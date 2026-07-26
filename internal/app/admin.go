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

package app

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/adminauth"
	admin "git.happydns.org/happyDomain/internal/api-admin/route"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/web-admin"
)

type Admin struct {
	router *gin.Engine
	cfg    *happydns.Options
	srv    *http.Server
}

func NewAdmin(app *App) *Admin {
	if app.cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	router := gin.New()

	// Same rationale as the public engine: never let a caller pick the address
	// it is logged under. Requests arriving on the unix socket are always
	// trusted by gin, which is fine, that socket is already a privileged path.
	if err := setupTrustedProxies(router, app.cfg); err != nil {
		log.Fatalf("%s", err)
	}

	router.Use(gin.Logger(), gin.Recovery())

	// Prepare usecases (admin uses unrestricted provider access)
	app.usecases.providerAdmin = providerUC.NewService(app.store, nil)

	admin.DeclareRoutes(
		app.cfg,
		router,
		app.store,
		admin.Dependencies{
			AuthUser:              app.usecases.authUser,
			Domain:                app.usecases.domain,
			Provider:              app.usecases.providerAdmin,
			RemoteZoneImporter:    app.usecases.orchestrator.RemoteZoneImporter,
			Service:               app.usecases.service,
			User:                  app.usecases.user,
			Zone:                  app.usecases.zone,
			ZoneCorrectionApplier: app.usecases.orchestrator.ZoneCorrectionApplier,
			ZoneImporter:          app.usecases.orchestrator.ZoneImporter,
			ZoneService:           app.usecases.zoneService,
		},
	)
	web.DeclareRoutes(app.cfg, router)

	return &Admin{
		router: router,
		cfg:    app.cfg,
	}
}

func (app *Admin) Start() {
	isTCP := app.cfg.HasNetworkAdminBind()

	// Refuse to expose an unauthenticated admin interface over the network: a
	// bind reachable from other hosts without a configured password would hand
	// the whole database to anyone who can reach the port. Only the admin
	// listener is given up here: the public API runs in the same process and
	// must keep serving, so this must never call log.Fatal.
	if isTCP && app.cfg.AdminPasswordHash == "" {
		if !app.cfg.HasLoopbackAdminBind() {
			log.Printf("ERROR: the admin interface is NOT started: it is bound to the network address %q but no admin password is configured. Set HAPPYDOMAIN_ADMIN_PASSWORD_HASH (see `happydomain admin-hash`), bind it to 127.0.0.1 behind an authenticating reverse proxy, or bind it to a local unix socket instead.", app.cfg.AdminBind)
			return
		}

		// A loopback bind is only reachable from the machine itself, which is
		// the historical single-host deployment (often fronted by a proxy that
		// does the authentication). Keep it working, but be loud about it.
		log.Printf("WARNING: the admin interface is bound to %q without an admin password: anyone able to reach that port, including any local user, has full administrative access. Set HAPPYDOMAIN_ADMIN_PASSWORD_HASH (see `happydomain admin-hash`).", app.cfg.AdminBind)
	}

	if app.cfg.AdminPasswordHash != "" && !adminauth.IsHashed(app.cfg.AdminPasswordHash) {
		if adminauth.IsMalformedHash(app.cfg.AdminPasswordHash) {
			log.Println("WARNING: the admin password starts with the bcrypt prefix `$2` but is not a valid bcrypt hash; it is used as a cleartext password. If you meant to configure a hash, it got truncated or mangled: generate a new one with `happydomain admin-hash`.")
		} else {
			log.Println("WARNING: the admin password is configured in cleartext; generate a hash with `happydomain admin-hash` and use it instead.")
		}
	}

	app.srv = &http.Server{
		Addr:              app.cfg.AdminBind,
		Handler:           app.router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	log.Printf("Admin interface listening on %s\n", app.cfg.AdminBind)
	if !isTCP {
		if _, err := os.Stat(app.cfg.AdminBind); !os.IsNotExist(err) {
			if err := os.Remove(app.cfg.AdminBind); err != nil {
				log.Fatal(err)
			}
		}

		unixListener, err := net.Listen("unix", app.cfg.AdminBind)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(app.srv.Serve(unixListener))
	} else if err := app.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("admin listen: %s\n", err)
	}
}
func (app *Admin) Stop() {
	// Start() can return before creating the server (refused bind), in which
	// case there is nothing to shut down.
	if app.srv == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.srv.Shutdown(ctx); err != nil {
		log.Fatal("Admin Server Shutdown:", err)
	}
}
