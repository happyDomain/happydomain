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
	"log"
	"time"

	"git.happydns.org/happyDomain/internal/captcha"
	"git.happydns.org/happyDomain/internal/mailer"
	"git.happydns.org/happyDomain/internal/metrics"
	"git.happydns.org/happyDomain/internal/newsletter"
	"git.happydns.org/happyDomain/internal/storage"
)

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
		m := &mailer.Mailer{
			MailFrom:   &app.cfg.MailFrom,
			SendMethod: mailer.NewSMTPMailer(app.cfg.MailSMTPHost, app.cfg.MailSMTPPort, app.cfg.MailSMTPUsername, app.cfg.MailSMTPPassword),
		}

		if app.cfg.MailSMTPTLSSNoVerify {
			m.SendMethod.(*mailer.SMTPMailer).WithTLSNoVerify()
		}
		app.mailer = m
	} else if !app.cfg.NoMail {
		app.mailer = &mailer.Mailer{
			MailFrom:   &app.cfg.MailFrom,
			SendMethod: &mailer.SystemSendmail{},
		}
	} else {
		app.mailer = &mailer.LogMailer{}
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

		metrics.NewStorageStatsCollector(storage.NewStatsProvider(app.store))
		app.store = newInstrumentedStorage(app.store)
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
