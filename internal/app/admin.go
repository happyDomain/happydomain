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
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	admin "git.happydns.org/happyDomain/internal/api-admin/route"
	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/usecase"
)

type Admin struct {
	router *gin.Engine
	cfg    *config.Options
	srv    *http.Server
}

func NewAdmin(app *App) *Admin {
	if app.cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Prepare usecases
	app.usecases.providerAdmin = usecase.NewAdminProviderUsecase(app.store)

	admin.DeclareRoutes(app.cfg, router, app.store, app)

	return &Admin{
		router: router,
		cfg:    app.cfg,
	}
}

func (app *Admin) Start() {
	app.srv = &http.Server{
		Addr:              app.cfg.AdminBind,
		Handler:           app.router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	log.Printf("Admin interface listening on %s\n", app.cfg.AdminBind)
	if !strings.Contains(app.cfg.AdminBind, ":") {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.srv.Shutdown(ctx); err != nil {
		log.Fatal("Admin Server Shutdown:", err)
	}
}
