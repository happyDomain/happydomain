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

// Package app wires together storage, usecases, transport and lifecycle for
// the happyDomain server. The bootstrap flow is split across several files:
//
//   - app.go        — App/Usecases types and the NewApp constructors
//   - init.go       — small init helpers (storage, mailer, newsletter, etc.)
//   - usecases.go   — initUsecases: the dependency-injection graph
//   - router.go     — gin router setup and route registration
//   - lifecycle.go  — Start / Stop (HTTP server + background workers)

package app

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/internal/usecase/orchestrator"

	"git.happydns.org/happyDomain/internal/captcha"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type Usecases struct {
	backup           happydns.BackupUsecase
	authentication   happydns.AuthenticationUsecase
	authUser         happydns.AuthUserUsecase
	authUserAdmin    happydns.AdminAuthUserUsecase
	domain           happydns.DomainUsecase
	domainAdmin      happydns.AdminDomainUsecase
	domainInfo       happydns.DomainInfoUsecase
	domainLog        happydns.DomainLogUsecase
	emailAutoconfig  happydns.EmailAutoconfigUsecase
	provider         happydns.ProviderUsecase
	providerAdmin    happydns.ProviderUsecase
	providerSpecs    happydns.ProviderSpecsUsecase
	providerSettings happydns.ProviderSettingsUsecase
	resolver         happydns.ResolverUsecase
	session          happydns.SessionUsecase
	service          happydns.ServiceUsecase
	serviceSpecs     happydns.ServiceSpecsUsecase
	user             happydns.UserUsecase
	userAdmin        happydns.AdminUserUsecase
	zone             happydns.ZoneUsecase
	zoneService      happydns.ZoneServiceUsecase

	orchestrator *orchestrator.Orchestrator

	checkerEngine    happydns.CheckerEngine
	checkerOptionsUC *checkerUC.CheckerOptionsUsecase
	checkerPlanUC    *checkerUC.CheckPlanUsecase
	checkerStatusUC  *checkerUC.CheckStatusUsecase
	checkerScheduler *checkerUC.Scheduler
	checkerJanitor   *checkerUC.Janitor
	checkerUserGater *checkerUC.UserGater

	notificationDispatcher *notifUC.Dispatcher
	notificationRegistry   *notifPkg.Registry
}

type App struct {
	captchaVerifier happydns.CaptchaVerifier
	cfg             *happydns.Options
	failureTracker  *captcha.FailureTracker
	insights        *insightsCollector
	mailer          happydns.Mailer
	newsletter      happydns.NewsletterSubscriptor
	router          *gin.Engine
	srv             *http.Server
	store           storage.Storage
	usecases        Usecases
}

func NewApp(cfg *happydns.Options) *App {
	app := &App{
		cfg: cfg,
	}

	app.initMailer()
	app.initStorageEngine()
	app.initNewsletter()
	app.initInsights()
	if err := app.initPlugins(); err != nil {
		log.Fatalf("Plugin initialization error: %s", err)
	}
	app.initUsecases()
	app.initCaptcha()
	app.setupRouter()

	return app
}

func NewAppWithStorage(cfg *happydns.Options, store storage.Storage) *App {
	app := &App{
		cfg:   cfg,
		store: store,
	}

	app.initMailer()
	app.initNewsletter()
	if err := app.initPlugins(); err != nil {
		log.Fatalf("Plugin initialization error: %s", err)
	}
	app.initUsecases()
	app.initCaptcha()
	app.setupRouter()

	return app
}
