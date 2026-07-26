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
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"slices"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	api "git.happydns.org/happyDomain/internal/api/route"
	"git.happydns.org/happyDomain/internal/metrics"
	"git.happydns.org/happyDomain/internal/session"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/web"
)

// setupTrustedProxies restricts which peers may set client IP headers. Without
// it gin trusts every peer, so c.ClientIP() returns whatever X-Forwarded-For
// the caller sends and anyone can reset the per-source rate limiters and the
// login lockout at will. An empty list (the default) trusts no proxy: callers
// are identified by their socket address.
func setupTrustedProxies(router *gin.Engine, cfg *happydns.Options) error {
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return fmt.Errorf("invalid -trusted-proxy value: %w", err)
	}

	return nil
}

// untrustedForwardedWarnInterval is how often warnUntrustedForwardedHeaders
// reports discarded client IP headers, so a flood of forged requests cannot
// flood the logs.
const untrustedForwardedWarnInterval = 15 * time.Minute

// trustedProxyPrefixes turns the configured trusted proxies into prefixes the
// warning middleware can test a peer against directly. The entries were
// already validated and canonicalized by the config layer, so anything that
// fails to parse here is simply not matched rather than reported twice.
func trustedProxyPrefixes(cfg *happydns.Options) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(cfg.TrustedProxies))

	for _, entry := range cfg.TrustedProxies {
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			prefixes = append(prefixes, prefix)
		} else if addr, err := netip.ParseAddr(entry); err == nil {
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
		}
	}

	return prefixes
}

// warnUntrustedForwardedHeaders logs when a peer that is not a trusted proxy
// sends a client IP header. Either a client is trying to spoof its address, or
// the operator put happyDomain behind a reverse proxy without declaring it
// with -trusted-proxy. The header value is deliberately never logged, it is
// attacker controlled.
//
// The peer is tested against the configured prefixes rather than inferred from
// ClientIP() falling back to the peer address: a trusted proxy also produces
// that fallback whenever it forwards a header gin cannot parse, or a request
// that originated on the proxy host itself, and warning there would push the
// operator to widen a trust list that is already correct.
func warnUntrustedForwardedHeaders(cfg *happydns.Options) gin.HandlerFunc {
	trusted := trustedProxyPrefixes(cfg)

	// Unix nanos of the last warning, as an atomic so that the common case (a
	// flood of forged headers, or a whole deployment behind an undeclared
	// proxy) costs a single load and never serializes requests on a mutex.
	var lastWarn atomic.Int64

	return func(c *gin.Context) {
		if c.GetHeader("X-Forwarded-For") != "" || c.GetHeader("X-Real-IP") != "" {
			if addrPort, err := netip.ParseAddrPort(c.Request.RemoteAddr); err == nil {
				peer := addrPort.Addr().Unmap()

				if !slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(peer) }) {
					warnUntrustedPeer(&lastWarn, peer)
				}
			}
		}

		c.Next()
	}
}

// warnUntrustedPeer emits the warning for peer unless one was already emitted
// less than untrustedForwardedWarnInterval ago.
func warnUntrustedPeer(lastWarn *atomic.Int64, peer netip.Addr) {
	previous := lastWarn.Load()

	now := time.Now().UnixNano()
	if now-previous <= int64(untrustedForwardedWarnInterval) {
		return
	}

	// Lost the race against a concurrent warning: that one covers this window.
	if !lastWarn.CompareAndSwap(previous, now) {
		return
	}

	log.Printf("%s: ignoring client IP header sent by an untrusted peer. If this address is your reverse proxy, declare it with -trusted-proxy (see docs/reverse-proxy.md); otherwise a client is trying to spoof its address.", peer)
}

func (app *App) setupRouter() {
	if app.cfg.DevProxy == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	gin.ForceConsoleColor()
	app.router = gin.New()

	if err := setupTrustedProxies(app.router, app.cfg); err != nil {
		log.Fatalf("%s", err)
	}

	app.router.Use(gin.Logger(), gin.Recovery(), warnUntrustedForwardedHeaders(app.cfg), metrics.HTTPMiddleware(), sessions.Sessions(
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
