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
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/storage"
	"git.happydns.org/happyDomain/ui"
)

type App struct {
	router *gin.Engine
	cfg    *config.Options
	srv    *http.Server
}

func NewApp(cfg *config.Options) App {
	if cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(sessions.Sessions(session.COOKIE_NAME, session.NewSessionStore(cfg, storage.MainStore, []byte(cfg.JWTSecretKey))))

	api.DeclareRoutes(cfg, router)
	ui.DeclareRoutes(cfg, router)

	if config.OIDCProviderURL != "" {
		authRoutes := router.Group("/auth")
		InitializeOIDC(cfg, authRoutes)
	}

	app := App{
		router: router,
		cfg:    cfg,
	}

	return app
}

func (app *App) Start() {
	app.srv = &http.Server{
		Addr:              app.cfg.Bind,
		Handler:           app.router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	log.Printf("Public interface listening on %s\n", app.cfg.Bind)
	if err := app.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}
func (app *App) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
}
