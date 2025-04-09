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

	api "git.happydns.org/happyDomain/api/route"
	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/mailer"
	"git.happydns.org/happyDomain/internal/newsletter"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/ui"
	"git.happydns.org/happyDomain/usecase"
)

type App struct {
	cfg    *config.Options
	mailer *mailer.Mailer
	router *gin.Engine
	srv    *http.Server
	store  storage.Storage

	AuthenticationService   happydns.AuthenticationUsecase
	AuthUserService         happydns.AuthUserUsecase
	DomainService           happydns.DomainUsecase
	DomainLogService        happydns.DomainLogUsecase
	ProviderService         happydns.ProviderUsecase
	ProviderServiceAdmin    happydns.ProviderUsecase
	ProviderSpecsService    happydns.ProviderSpecsUsecase
	ProviderSettingsService happydns.ProviderSettingsUsecase
	ResolverService         happydns.ResolverUsecase
	SessionService          happydns.SessionUsecase
	ServiceService          happydns.ServiceUsecase
	ServiceSpecsService     happydns.ServiceSpecsUsecase
	UserService             happydns.UserUsecase
	ZoneService             happydns.ZoneUsecase
}

func (a *App) GetAuthenticationService() happydns.AuthenticationUsecase {
	return a.AuthenticationService
}

func (a *App) GetAuthUserService() happydns.AuthUserUsecase {
	return a.AuthUserService
}

func (a *App) GetDomainService() happydns.DomainUsecase {
	return a.DomainService
}

func (a *App) GetDomainLogService() happydns.DomainLogUsecase {
	return a.DomainLogService
}

func (a *App) GetProviderService(secure bool) happydns.ProviderUsecase {
	if secure {
		return a.ProviderService
	} else {
		return a.ProviderServiceAdmin
	}
}

func (a *App) GetProviderSettingsService() happydns.ProviderSettingsUsecase {
	return a.ProviderSettingsService
}

func (a *App) GetProviderSpecsService() happydns.ProviderSpecsUsecase {
	return a.ProviderSpecsService
}

func (a *App) GetResolverService() happydns.ResolverUsecase {
	return a.ResolverService
}

func (a *App) GetServiceService() happydns.ServiceUsecase {
	return a.ServiceService
}

func (a *App) GetServiceSpecsService() happydns.ServiceSpecsUsecase {
	return a.ServiceSpecsService
}

func (a *App) GetSessionService() happydns.SessionUsecase {
	return a.SessionService
}

func (a *App) GetUserService() happydns.UserUsecase {
	return a.UserService
}

func (a *App) GetZoneService() happydns.ZoneUsecase {
	return a.ZoneService
}

func NewApp(cfg *config.Options) *App {
	app := &App{
		cfg: cfg,
	}

	// Initialize mailer
	if cfg.MailSMTPHost != "" {
		app.mailer = &mailer.Mailer{
			MailFrom:   &cfg.MailFrom,
			SendMethod: mailer.NewSMTPMailer(cfg.MailSMTPHost, cfg.MailSMTPPort, cfg.MailSMTPUsername, cfg.MailSMTPPassword),
		}

		if cfg.MailSMTPTLSSNoVerify {
			app.mailer.SendMethod.(*mailer.SMTPMailer).WithTLSNoVerify()
		}

	} else if !cfg.NoMail {
		app.mailer = &mailer.Mailer{
			MailFrom:   &cfg.MailFrom,
			SendMethod: &mailer.SystemSendmail{},
		}
	}

	// Initialize storage
	if s, ok := storage.StorageEngines[cfg.StorageEngine]; !ok {
		log.Fatalf("Nonexistent storage engine: %q, please select one of: %v", cfg.StorageEngine, storage.GetStorageEngines())
	} else {
		var err error
		log.Println("Opening database...")
		app.store, err = s()
		if err != nil {
			log.Fatal("Could not open the database: ", err)
		}

		log.Println("Performing database migrations...")
		if err = app.store.DoMigration(); err != nil {
			log.Fatal("Could not migrate database: ", err)
		}
	}

	// Initialize newsletter registration
	var ns happydns.NewsletterSubscriptor
	if cfg.ListmonkURL.URL != nil {
		ns = &newsletter.ListmonkNewsletterSubscription{
			ListmonkURL: cfg.ListmonkURL.URL,
			ListmonkId:  cfg.ListmonkId,
		}
	} else {
		ns = &newsletter.DummyNewsletterSubscription{}
	}

	// Prepare usecases
	app.ProviderSpecsService = usecase.NewProviderSpecsUsecase()
	app.ProviderSettingsService = usecase.NewProviderSettingsUsecase(cfg, app.store)
	app.ProviderService = usecase.NewProviderUsecase(cfg, app.store)
	app.ServiceService = usecase.NewServiceUsecase()
	app.ServiceSpecsService = usecase.NewServiceSpecsUsecase()
	app.ZoneService = usecase.NewZoneUsecase(app.ProviderService, app.ServiceService, app.store)
	app.DomainLogService = usecase.NewDomainLogUsecase(app.store)
	app.DomainService = usecase.NewDomainUsecase(app.store, app.DomainLogService, app.ProviderService, app.ZoneService)

	app.UserService = usecase.NewUserUsecase(app.store, ns)
	app.AuthenticationService = usecase.NewAuthenticationUsecase(cfg, app.store, app.UserService)
	app.AuthUserService = usecase.NewAuthUserUsecase(cfg, app.mailer, app.store)
	app.ResolverService = usecase.NewResolverUsecase(cfg)
	app.SessionService = usecase.NewSessionUsecase(app.store)

	// Initialize router
	if cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	app.router = gin.New()
	app.router.Use(gin.Logger(), gin.Recovery(), sessions.Sessions(
		session.COOKIE_NAME,
		session.NewSessionStore(cfg, app.store, []byte(cfg.JWTSecretKey)),
	))

	api.DeclareRoutes(cfg, app.router, app)
	ui.DeclareRoutes(cfg, app.router)

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

	// Close storage
	if app.store != nil {
		app.store.Close()
	}
}
