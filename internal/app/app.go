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
	cfg      *config.Options
	mailer   *mailer.Mailer
	router   *gin.Engine
	srv      *http.Server
	insights *insightsCollector
	store    storage.Storage

	AuthenticationUsecase   happydns.AuthenticationUsecase
	AuthUserUsecase         happydns.AuthUserUsecase
	DomainUsecase           happydns.DomainUsecase
	DomainLogUsecase        happydns.DomainLogUsecase
	ProviderUsecase         happydns.ProviderUsecase
	ProviderUsecaseAdmin    happydns.ProviderUsecase
	ProviderSpecsUsecase    happydns.ProviderSpecsUsecase
	ProviderSettingsUsecase happydns.ProviderSettingsUsecase
	ResolverUsecase         happydns.ResolverUsecase
	SessionUsecase          happydns.SessionUsecase
	ServiceUsecase          happydns.ServiceUsecase
	ServiceSpecsUsecase     happydns.ServiceSpecsUsecase
	UserUsecase             happydns.UserUsecase
	ZoneUsecase             happydns.ZoneUsecase
}

func (a *App) GetAuthenticationUsecase() happydns.AuthenticationUsecase {
	return a.AuthenticationUsecase
}

func (a *App) GetAuthUserUsecase() happydns.AuthUserUsecase {
	return a.AuthUserUsecase
}

func (a *App) GetDomainUsecase() happydns.DomainUsecase {
	return a.DomainUsecase
}

func (a *App) GetDomainLogUsecase() happydns.DomainLogUsecase {
	return a.DomainLogUsecase
}

func (a *App) GetProviderUsecase(secure bool) happydns.ProviderUsecase {
	if secure {
		return a.ProviderUsecase
	} else {
		return a.ProviderUsecaseAdmin
	}
}

func (a *App) GetProviderSettingsUsecase() happydns.ProviderSettingsUsecase {
	return a.ProviderSettingsUsecase
}

func (a *App) GetProviderSpecsUsecase() happydns.ProviderSpecsUsecase {
	return a.ProviderSpecsUsecase
}

func (a *App) GetResolverUsecase() happydns.ResolverUsecase {
	return a.ResolverUsecase
}

func (a *App) GetServiceUsecase() happydns.ServiceUsecase {
	return a.ServiceUsecase
}

func (a *App) GetServiceSpecsUsecase() happydns.ServiceSpecsUsecase {
	return a.ServiceSpecsUsecase
}

func (a *App) GetSessionUsecase() happydns.SessionUsecase {
	return a.SessionUsecase
}

func (a *App) GetUserUsecase() happydns.UserUsecase {
	return a.UserUsecase
}

func (a *App) GetZoneUsecase() happydns.ZoneUsecase {
	return a.ZoneUsecase
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
		if err = app.store.MigrateSchema(); err != nil {
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

	if !cfg.OptOutInsights {
		app.insights = &insightsCollector{
			cfg:   app.cfg,
			store: app.store,
			stop:  make(chan bool),
		}
	}

	// Prepare usecases
	app.ProviderSpecsUsecase = usecase.NewProviderSpecsUsecase()
	app.ProviderUsecase = usecase.NewProviderUsecase(cfg, app.store)
	app.ProviderSettingsUsecase = usecase.NewProviderSettingsUsecase(cfg, app.ProviderUsecase, app.store)
	app.ServiceUsecase = usecase.NewServiceUsecase()
	app.ServiceSpecsUsecase = usecase.NewServiceSpecsUsecase()
	app.ZoneUsecase = usecase.NewZoneUsecase(app.ProviderUsecase, app.ServiceUsecase, app.store)
	app.DomainLogUsecase = usecase.NewDomainLogUsecase(app.store)
	app.DomainUsecase = usecase.NewDomainUsecase(app.store, app.DomainLogUsecase, app.ProviderUsecase, app.ZoneUsecase)

	app.UserUsecase = usecase.NewUserUsecase(app.store, ns)
	app.AuthenticationUsecase = usecase.NewAuthenticationUsecase(cfg, app.store, app.UserUsecase)
	app.AuthUserUsecase = usecase.NewAuthUserUsecase(cfg, app.mailer, app.store)
	app.ResolverUsecase = usecase.NewResolverUsecase(cfg)
	app.SessionUsecase = usecase.NewSessionUsecase(app.store)

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
