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
	"git.happydns.org/happyDomain/internal/captcha"
	"git.happydns.org/happyDomain/internal/mailer"
	"git.happydns.org/happyDomain/internal/newsletter"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/usecase"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	checkUC "git.happydns.org/happyDomain/internal/usecase/check"
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
	checker          happydns.CheckerUsecase
	domain           happydns.DomainUsecase
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
	captchaVerifier happydns.CaptchaVerifier
	cfg             *happydns.Options
	failureTracker  *captcha.FailureTracker
	insights        *insightsCollector
	mailer          *mailer.Mailer
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

func (app *App) initCaptcha() {
	app.captchaVerifier = captcha.NewVerifier(app.cfg.CaptchaProvider)

	threshold := app.cfg.CaptchaLoginThreshold
	if threshold <= 0 {
		threshold = 3
	}

	app.failureTracker = captcha.NewFailureTracker(threshold, 15*time.Minute)
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
			ListmonkID:  app.cfg.ListmonkID,
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
			stop:  make(chan struct{}, 1),
		}
	}
}

func (app *App) initUsecases() {
	sessionService := sessionUC.NewService(app.store)
	authUserService := authuserUC.NewAuthUserUsecases(app.cfg, app.mailer, app.store, sessionService)
	domainLogService := domainlogUC.NewService(app.store)
	providerService := providerUC.NewRestrictedService(app.cfg, app.store)
	providerAdminService := providerUC.NewService(app.store)
	serviceService := serviceUC.NewServiceUsecases()
	zoneService := zoneUC.NewZoneUsecases(app.store, serviceService)

	app.usecases.providerSpecs = usecase.NewProviderSpecsUsecase()
	app.usecases.provider = providerService
	app.usecases.providerAdmin = providerAdminService
	app.usecases.providerSettings = usecase.NewProviderSettingsUsecase(app.cfg, app.usecases.provider, app.store)
	app.usecases.service = serviceService
	app.usecases.serviceSpecs = usecase.NewServiceSpecsUsecase()
	app.usecases.zone = zoneService
	app.usecases.domainLog = domainLogService

	domainService := domainUC.NewService(app.store, providerAdminService, zoneService.GetZoneUC, providerAdminService, domainLogService)
	app.usecases.domain = domainService
	app.usecases.zoneService = zoneServiceUC.NewZoneServiceUsecases(domainService, zoneService.CreateZoneUC, serviceService.ValidateServiceUC, app.store)

	app.usecases.user = userUC.NewUserUsecases(app.store, app.newsletter, authUserService, sessionService)
	app.usecases.authentication = usecase.NewAuthenticationUsecase(app.cfg, app.store, app.usecases.user)
	app.usecases.authUser = authUserService
	app.usecases.resolver = usecase.NewResolverUsecase(app.cfg)
	app.usecases.session = sessionService
	app.usecases.checker = checkUC.NewCheckerUsecase(app.cfg)

	app.usecases.orchestrator = orchestrator.NewOrchestrator(
		domainLogService,
		domainService,
		providerAdminService,
		zoneService.ListRecordsUC,
		providerAdminService,
		zoneService.CreateZoneUC,
		providerAdminService,
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

	if len(app.cfg.BasePath) > 0 {
		app.router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, app.cfg.BasePath)
		})
	}

	baserouter := app.router.Group(app.cfg.BasePath)

	api.DeclareRoutes(app.cfg, baserouter, api.Dependencies{
		Authentication:        app.usecases.authentication,
		AuthUser:              app.usecases.authUser,
		CaptchaVerifier:       app.captchaVerifier,
		Domain:                app.usecases.domain,
		DomainLog:             app.usecases.domainLog,
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
	})
	web.DeclareRoutes(app.cfg, baserouter, app.captchaVerifier)
	web.NoRoute(app.cfg, app.router)
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

	if app.failureTracker != nil {
		app.failureTracker.Close()
	}
}
