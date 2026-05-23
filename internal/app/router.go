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
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	api "git.happydns.org/happyDomain/internal/api/route"
	"git.happydns.org/happyDomain/internal/metrics"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/web"
)

func (app *App) setupRouter() {
	if app.cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	app.router = gin.New()
	app.router.Use(gin.Logger(), gin.Recovery(), metrics.HTTPMiddleware(), sessions.Sessions(
		session.COOKIE_NAME,
		session.NewSessionStore(app.cfg, app.store, []byte(app.cfg.JWTSecretKey)),
	))

	if len(app.cfg.BasePath) > 0 {
		app.router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, app.cfg.BasePath)
		})
	}

	baserouter := app.router.Group(app.cfg.BasePath)

	api.DeclareRoutes(
		app.cfg,
		baserouter,
		api.Dependencies{
			Backup:                app.usecases.backup,
			Authentication:        app.usecases.authentication,
			AuthUser:              app.usecases.authUser,
			CaptchaVerifier:       app.captchaVerifier,
			Domain:                app.usecases.domain,
			DomainInfo:            app.usecases.domainInfo,
			DomainLog:             app.usecases.domainLog,
			EmailAutoconfig:       app.usecases.emailAutoconfig,
			FailureTracker:        app.failureTracker,
			Provider:              app.usecases.provider,
			ProviderSettings:      app.usecases.providerSettings,
			ProviderSpecs:         app.usecases.providerSpecs,
			RemoteZoneImporter:    app.usecases.orchestrator.RemoteZoneImporter,
			Resolver:              app.usecases.resolver,
			Service:               app.usecases.service,
			ServiceSpecs:          app.usecases.serviceSpecs,
			Session:               app.usecases.session,
			User:                  app.usecases.user,
			Zone:                  app.usecases.zone,
			ZoneCorrectionApplier: app.usecases.orchestrator.ZoneCorrectionApplier,
			ZoneImporter:          app.usecases.orchestrator.ZoneImporter,
			ZoneService:           app.usecases.zoneService,

			CheckerEngine:       app.usecases.checkerEngine,
			CheckerOptionsUC:    app.usecases.checkerOptionsUC,
			CheckPlanUC:         app.usecases.checkerPlanUC,
			CheckStatusUC:       app.usecases.checkerStatusUC,
			PlannedProvider:     app.usecases.checkerScheduler,
			BudgetChecker:       app.usecases.checkerUserGater,
			CountManualTriggers: app.cfg.CheckerCountManualTriggers,

			NotificationDispatcher: app.usecases.notificationDispatcher,
			NotificationRegistry:   app.usecases.notificationRegistry,
			NotificationChannels:   app.store,
			NotificationPrefs:      app.store,
			NotificationRecords:    app.store,
		},
	)
	web.DeclareRoutes(app.cfg, baserouter, app.captchaVerifier)
	web.NoRoute(app.cfg, app.router)
}
