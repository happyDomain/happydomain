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
)

func (app *App) Start() {
	app.srv = &http.Server{
		Addr:              app.cfg.Bind,
		Handler:           app.router,
		ReadHeaderTimeout: 15 * time.Second,
	}

	if app.insights != nil {
		go app.insights.Run()
	}

	// Reconcile executions left "running" by a previous process that
	// crashed or was killed mid-run, before the scheduler starts queuing
	// new work.
	if recoverer, ok := app.usecases.checkerEngine.(interface {
		RecoverStaleExecutions(ctx context.Context) (int, error)
	}); ok {
		if n, err := recoverer.RecoverStaleExecutions(context.Background()); err != nil {
			log.Printf("CheckerEngine: failed to recover stale executions: %v", err)
		} else if n > 0 {
			log.Printf("CheckerEngine: recovered %d stale execution(s) from previous run", n)
		}
	}

	if app.usecases.checkerScheduler != nil && !app.cfg.DisableCheckerScheduler {
		app.usecases.checkerScheduler.Start(context.Background())
	}

	if app.usecases.checkerJanitor != nil {
		app.usecases.checkerJanitor.Start(context.Background())
	}

	if app.usecases.checkerUserGater != nil {
		app.usecases.checkerUserGater.Start(context.Background())
	}

	if app.usecases.notificationDispatcher != nil {
		app.usecases.notificationDispatcher.Start()
	}

	log.Printf("Public interface listening on %s\n", app.cfg.Bind)
	if err := app.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}

func (app *App) Stop() {
	// Stop background workers first so they don't dispatch new work while
	// the HTTP server is draining. Each Stop() cancels its context and
	// waits for in-flight goroutines to return.
	if app.usecases.checkerScheduler != nil {
		app.usecases.checkerScheduler.Stop()
	}

	if app.usecases.checkerJanitor != nil {
		app.usecases.checkerJanitor.Stop()
	}

	if app.usecases.checkerUserGater != nil {
		app.usecases.checkerUserGater.Stop()
	}

	// Drain in-flight notification sends after the scheduler is stopped
	// so no new jobs can be enqueued while we wait.
	if app.usecases.notificationDispatcher != nil {
		app.usecases.notificationDispatcher.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.srv.Shutdown(ctx); err != nil {
		// Don't log.Fatal here: that would skip the storage/insights
		// cleanup below and risk leaving state on disk inconsistent.
		log.Printf("Server Shutdown: %v", err)
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
