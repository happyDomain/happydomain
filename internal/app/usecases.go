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
	checkerPkg "git.happydns.org/happyDomain/internal/dnschecker"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/usecase"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	backupUC "git.happydns.org/happyDomain/internal/usecase/backup"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	domainUC "git.happydns.org/happyDomain/internal/usecase/domain"
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	emailAutoconfigUC "git.happydns.org/happyDomain/internal/usecase/emailautoconfig"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	userUC "git.happydns.org/happyDomain/internal/usecase/user"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	zoneServiceUC "git.happydns.org/happyDomain/internal/usecase/zone_service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
	"git.happydns.org/happyDomain/services/abstract"
)

func (app *App) initUsecases() {
	sessionService := sessionUC.NewService(app.store)
	authUserService := authuserUC.NewAuthUserUsecases(
		app.cfg,
		app.mailer,
		app.store,
		sessionService,
	)
	domainLogService := domainlogUC.NewService(app.store)
	providerService := providerUC.NewRestrictedService(app.cfg, app.store)
	providerAdminService := providerUC.NewService(app.store, nil)
	serviceService := serviceUC.NewServiceUsecases()
	zoneService := zoneUC.NewZoneUsecases(app.store, serviceService)

	app.usecases.backup = backupUC.NewUsecase(app.store)
	app.usecases.providerSpecs = usecase.NewProviderSpecsUsecase()
	app.usecases.provider = providerService
	app.usecases.providerAdmin = providerAdminService
	app.usecases.providerSettings = usecase.NewProviderSettingsUsecase(app.cfg, app.usecases.provider)
	app.usecases.service = serviceService
	app.usecases.serviceSpecs = usecase.NewServiceSpecsUsecase()
	app.usecases.zone = zoneService
	app.usecases.domainInfo = usecase.NewDomainInfoUsecase(
		domaininfo.GetDomainRDAPInfo,
		domaininfo.GetDomainWhoisInfo,
	)
	app.usecases.domainLog = domainLogService

	// Email auto-configuration: derive the autoconfig CNAME target from
	// MailAutoconfigHost (if set) or fall back to ExternalURL.Host.
	autoconfigHost := app.cfg.MailAutoconfigHost
	if autoconfigHost == "" {
		autoconfigHost = app.cfg.ExternalURL.Hostname()
	}
	abstract.SetAutoconfigHost(autoconfigHost)
	app.usecases.emailAutoconfig = emailAutoconfigUC.NewUsecase(app.store, zoneService.GetZoneUC)

	domainService := domainUC.NewService(
		app.store,
		providerAdminService,
		zoneService.GetZoneUC,
		providerAdminService,
		domainLogService,
	)
	app.usecases.domain = domainService
	app.usecases.domainAdmin = domainService
	app.usecases.zoneService = zoneServiceUC.NewZoneServiceUsecases(
		domainService,
		zoneService.CreateZoneUC,
		serviceService.ValidateServiceUC,
		app.store,
	)

	userService := userUC.NewUserUsecases(
		app.store,
		app.newsletter,
		authUserService,
		sessionService,
	)
	app.usecases.user = userService
	app.usecases.userAdmin = userService
	app.usecases.authentication = usecase.NewAuthenticationUsecase(app.cfg, app.store, app.usecases.user)
	app.usecases.authUser = authUserService
	app.usecases.authUserAdmin = authUserService
	app.usecases.resolver = usecase.NewResolverUsecase(app.cfg)
	app.usecases.session = sessionService

	app.usecases.orchestrator = orchestrator.NewOrchestrator(
		domainLogService,
		domainService,
		providerAdminService,
		zoneService.ListRecordsUC,
		providerAdminService,
		zoneService.CreateZoneUC,
		zoneService.GetZoneUC,
		providerAdminService,
		zoneService.UpdateZoneUC,
	)

	// Checker system.
	checkerPkg.SetHTTPTimeout(app.cfg.CheckerHTTPTimeout)
	app.usecases.checkerOptionsUC = checkerUC.NewCheckerOptionsUsecase(app.store, app.store).
		WithDiscoveryEntryStore(app.store).
		WithAdminOptions(app.cfg.CheckerAdminOptions)
	app.usecases.checkerPlanUC = checkerUC.NewCheckPlanUsecase(app.store)
	app.usecases.checkerStatusUC = checkerUC.NewCheckStatusUsecase(app.store, app.store, app.store, app.store, app.usecases.checkerOptionsUC)
	app.usecases.checkerEngine = checkerUC.NewCheckerEngine(
		app.usecases.checkerOptionsUC,
		app.store,
		app.store,
		app.store,
		app.store,
		app.store,
		app.store,
	)
	// Build the user-level gate so paused or long-inactive users do not
	// get checked. The same user resolver is reused by the janitor for
	// per-user retention overrides.
	app.usecases.checkerUserGater = checkerUC.NewUserGater(app.store, app.cfg.CheckerInactivityPauseDays, app.cfg.CheckerMaxChecksPerDay)
	app.usecases.checkerScheduler = checkerUC.NewScheduler(
		app.usecases.checkerEngine,
		app.cfg.CheckerMaxConcurrency,
		app.store, app.store, app.store, app.store,
		app.usecases.checkerUserGater.AllowWithInterval,
		app.usecases.checkerUserGater.IncrementUsage,
	)

	// Invalidate the scheduler's user gate cache whenever a user is updated
	// (e.g. login refreshing LastSeen, admin toggling SchedulingPaused).
	userService.SetOnUserChanged(func(id happydns.Identifier) {
		app.usecases.checkerUserGater.Invalidate(id.String())
	})

	// Retention janitor.
	app.usecases.checkerJanitor = checkerUC.NewJanitor(
		app.store,
		app.store,
		app.store,
		app.store,
		app.store,
		checkerUC.DefaultRetentionPolicy(app.cfg.CheckerRetentionDays),
		app.cfg.CheckerJanitorInterval,
	)

	// Wire scheduler notifications for incremental queue updates.
	domainService.SetSchedulerNotifier(app.usecases.checkerScheduler)
	app.usecases.orchestrator.SetSchedulerNotifier(app.usecases.checkerScheduler)

	// Notification system: dispatcher fans out checker results to user
	// channels (email/webhook/UnifiedPush) based on per-target preferences.
	baseURL := app.cfg.GetBaseURL()
	registry := notifPkg.NewRegistry()
	registry.Register(notifPkg.Adapt(notifPkg.NewEmailSender(app.mailer, baseURL)))
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender(baseURL)))
	registry.Register(notifPkg.Adapt(notifPkg.NewUnifiedPushSender(baseURL)))
	app.usecases.notificationRegistry = registry
	resolver := notifUC.NewResolver(app.store, app.store)
	pool := notifUC.NewPool(registry, app.store)
	tester := notifUC.NewTester(registry)
	stateLocker := notifUC.NewStateLocker()
	ack := notifUC.NewAckService(app.store, stateLocker)
	app.usecases.notificationDispatcher = notifUC.NewDispatcher(
		app.store,
		app.store,
		app.store,
		app.store,
		resolver,
		pool,
		tester,
		ack,
		stateLocker,
	)
	if cb, ok := app.usecases.checkerEngine.(checkerUC.ExecutionCallbackSetter); ok {
		cb.SetExecutionCallback(app.usecases.notificationDispatcher.OnExecutionComplete)
	}
}
