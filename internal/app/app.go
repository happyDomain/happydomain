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

	api "git.happydns.org/happyDomain/internal/api/route"
	"git.happydns.org/happyDomain/internal/mailer"
	"git.happydns.org/happyDomain/internal/newsletter"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/usecase"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	domainUC "git.happydns.org/happyDomain/internal/usecase/domain"
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	userUC "git.happydns.org/happyDomain/internal/usecase/user"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	zoneServiceUC "git.happydns.org/happyDomain/internal/usecase/zone_service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/web"
)

type Usecases struct {
	authentication   happydns.AuthenticationUsecase
	authUser         happydns.AuthUserUsecase
	domain           happydns.DomainUsecase
	domainInfo       happydns.DomainInfoUsecase
	domainLog        happydns.DomainLogUsecase
	provider         happydns.ProviderUsecase
	providerAdmin    happydns.ProviderUsecase
	providerSpecs    happydns.ProviderSpecsUsecase
	providerSettings happydns.ProviderSettingsUsecase
	resolver         happydns.ResolverUsecase
	session          happydns.SessionUsecase
	service          happydns.ServiceUsecase
	serviceSpecs     happydns.ServiceSpecsUsecase
	user             happydns.UserUsecase
	zone             happydns.ZoneUsecase
	zoneService      happydns.ZoneServiceUsecase

	orchestrator *orchestrator.Orchestrator
}

type App struct {
	cfg        *happydns.Options
	mailer     *mailer.Mailer
	newsletter happydns.NewsletterSubscriptor
	router     *gin.Engine
	srv        *http.Server
	insights   *insightsCollector
	store      storage.Storage
	usecases   Usecases
}

func (a *App) AuthenticationUsecase() happydns.AuthenticationUsecase {
	return a.usecases.authentication
}

func (a *App) AuthUserUsecase() happydns.AuthUserUsecase {
	return a.usecases.authUser
}

func (a *App) DomainUsecase() happydns.DomainUsecase {
	return a.usecases.domain
}

func (a *App) DomainInfoUsecase() happydns.DomainInfoUsecase {
	return a.usecases.domainInfo
}

func (a *App) DomainLogUsecase() happydns.DomainLogUsecase {
	return a.usecases.domainLog
}

func (a *App) Orchestrator() *orchestrator.Orchestrator {
	return a.usecases.orchestrator
}

func (a *App) ProviderUsecase(secure bool) happydns.ProviderUsecase {
	if secure {
		return a.usecases.provider
	} else {
		return a.usecases.providerAdmin
	}
}

func (a *App) ProviderSettingsUsecase() happydns.ProviderSettingsUsecase {
	return a.usecases.providerSettings
}

func (a *App) ProviderSpecsUsecase() happydns.ProviderSpecsUsecase {
	return a.usecases.providerSpecs
}

func (a *App) ResolverUsecase() happydns.ResolverUsecase {
	return a.usecases.resolver
}

func (a *App) RemoteZoneImporterUsecase() happydns.RemoteZoneImporterUsecase {
	return a.usecases.orchestrator.RemoteZoneImporter
}

func (a *App) ServiceUsecase() happydns.ServiceUsecase {
	return a.usecases.service
}

func (a *App) ServiceSpecsUsecase() happydns.ServiceSpecsUsecase {
	return a.usecases.serviceSpecs
}

func (a *App) SessionUsecase() happydns.SessionUsecase {
	return a.usecases.session
}

func (a *App) UserUsecase() happydns.UserUsecase {
	return a.usecases.user
}

func (a *App) ZoneCorrectionApplierUsecase() happydns.ZoneCorrectionApplierUsecase {
	return a.usecases.orchestrator.ZoneCorrectionApplier
}

func (a *App) ZoneImporterUsecase() happydns.ZoneImporterUsecase {
	return a.usecases.orchestrator.ZoneImporter
}

func (a *App) ZoneUsecase() happydns.ZoneUsecase {
	return a.usecases.zone
}

func (a *App) ZoneServiceUsecase() happydns.ZoneServiceUsecase {
	return a.usecases.zoneService
}

func NewApp(cfg *happydns.Options) *App {
	app := &App{
		cfg: cfg,
	}

	app.initMailer()
	app.initStorageEngine()
	app.initNewsletter()
	app.initInsights()
	app.initUsecases()
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
	app.initUsecases()
	app.setupRouter()

	return app
}

func (app *App) initMailer() {
	if app.cfg.MailSMTPHost != "" {
		app.mailer = &mailer.Mailer{
			MailFrom:   &app.cfg.MailFrom,
			SendMethod: mailer.NewSMTPMailer(app.cfg.MailSMTPHost, app.cfg.MailSMTPPort, app.cfg.MailSMTPUsername, app.cfg.MailSMTPPassword),
		}

		if app.cfg.MailSMTPTLSSNoVerify {
			app.mailer.SendMethod.(*mailer.SMTPMailer).WithTLSNoVerify()
		}
	} else if !app.cfg.NoMail {
		app.mailer = &mailer.Mailer{
			MailFrom:   &app.cfg.MailFrom,
			SendMethod: &mailer.SystemSendmail{},
		}
	}
}

func (app *App) initStorageEngine() {
	if s, ok := storage.StorageEngines[app.cfg.StorageEngine]; !ok {
		log.Fatalf("Nonexistent storage engine: %q, please select one of: %v", app.cfg.StorageEngine, storage.GetStorageEngines())
	} else {
		var err error
		log.Println("Opening database...")
		app.store, err = s()
		if err != nil {
			log.Fatal("Could not open the database: ", err)
		}

		log.Println("Performing database migrations...")
		if err = app.store.MigrateSchema(); err != nil {
			log.Fatal("Could not migrate database: ", err)
		}
	}
}

func (app *App) initNewsletter() {
	if app.cfg.ListmonkURL.String() != "" {
		app.newsletter = &newsletter.ListmonkNewsletterSubscription{
			ListmonkURL: &app.cfg.ListmonkURL,
			ListmonkId:  app.cfg.ListmonkId,
		}
	} else {
		app.newsletter = &newsletter.DummyNewsletterSubscription{}
	}
}

func (app *App) initInsights() {
	if !app.cfg.OptOutInsights {
		app.insights = &insightsCollector{
			cfg:   app.cfg,
			store: app.store,
			stop:  make(chan bool),
		}
	}
}

func (app *App) initUsecases() {
	sessionService := sessionUC.NewSessionUsecases(app.store)
	authUserService := authuserUC.NewAuthUserUsecases(app.cfg, app.mailer, app.store, sessionService.CloseUserSessionsUC)
	domainLogService := domainlogUC.NewDomainLogUsecases(app.store)
	providerService := providerUC.NewRestrictedProviderUsecases(app.cfg, app.store)
	serviceService := serviceUC.NewServiceUsecases()
	zoneService := zoneUC.NewZoneUsecases(app.store, serviceService)

	app.usecases.providerSpecs = usecase.NewProviderSpecsUsecase()
	app.usecases.provider = providerService
	app.usecases.providerSettings = usecase.NewProviderSettingsUsecase(app.cfg, app.usecases.provider, app.store)
	app.usecases.service = serviceService
	app.usecases.serviceSpecs = usecase.NewServiceSpecsUsecase()
	app.usecases.zone = zoneService
	app.usecases.domainInfo = usecase.NewDomainInfoUsecase()
	app.usecases.domainLog = domainLogService

	domainService := domainUC.NewDomainUsecases(app.store, providerService.GetProviderUC, zoneService.GetZoneUC, providerService.DomainExistenceUC, domainLogService.CreateDomainLogUC)
	app.usecases.domain = domainService
	app.usecases.zoneService = zoneServiceUC.NewZoneServiceUsecases(domainService.UpdateDomainUC, zoneService.CreateZoneUC, serviceService.ValidateServiceUC, app.store)

	app.usecases.user = userUC.NewUserUsecases(app.store, app.newsletter, authUserService.GetAuthUserUC, sessionService.CloseUserSessionsUC)
	app.usecases.authentication = usecase.NewAuthenticationUsecase(app.cfg, app.store, app.usecases.user)
	app.usecases.authUser = authUserService
	app.usecases.resolver = usecase.NewResolverUsecase(app.cfg)
	app.usecases.session = sessionService

	app.usecases.orchestrator = orchestrator.NewOrchestrator(
		domainLogService.CreateDomainLogUC,
		domainService.UpdateDomainUC,
		providerService.GetProviderUC,
		zoneService.ListRecordsUC,
		providerService.ZoneCorrectionsUC,
		zoneService.CreateZoneUC,
		providerService.RetrieveZoneUC,
		zoneService.UpdateZoneUC,
	)
}

func (app *App) setupRouter() {
	if app.cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	app.router = gin.New()
	app.router.Use(gin.Logger(), gin.Recovery(), sessions.Sessions(
		session.COOKIE_NAME,
		session.NewSessionStore(app.cfg, app.store, []byte(app.cfg.JWTSecretKey)),
	))

	api.DeclareRoutes(app.cfg, app.router, app)
	web.DeclareRoutes(app.cfg, app.router)
}

func (app *App) Start() {
	app.srv = &http.Server{
		Addr:              app.cfg.Bind,
		Handler:           app.router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	if app.insights != nil {
		go app.insights.Run()
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

	// Close storage
	if app.store != nil {
		app.store.Close()
	}

	if app.insights != nil {
		app.insights.Close()
	}
}
