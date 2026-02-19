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

package route

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	happydns "git.happydns.org/happyDomain/model"
)

// Dependencies holds all use cases required to register the public API routes.
// It is a plain struct — no methods, no interface — constructed once in app.go.
type Dependencies struct {
	Authentication        happydns.AuthenticationUsecase
	AuthUser              happydns.AuthUserUsecase
	CaptchaVerifier       happydns.CaptchaVerifier
	Domain                happydns.DomainUsecase
	DomainLog             happydns.DomainLogUsecase
	FailureTracker        happydns.FailureTracker
	Provider              happydns.ProviderUsecase
	ProviderSettings      happydns.ProviderSettingsUsecase
	ProviderSpecs         happydns.ProviderSpecsUsecase
	RemoteZoneImporter    happydns.RemoteZoneImporterUsecase
	Resolver              happydns.ResolverUsecase
	Service               happydns.ServiceUsecase
	ServiceSpecs          happydns.ServiceSpecsUsecase
	Session               happydns.SessionUsecase
	User                  happydns.UserUsecase
	Zone                  happydns.ZoneUsecase
	ZoneCorrectionApplier happydns.ZoneCorrectionApplierUsecase
	ZoneImporter          happydns.ZoneImporterUsecase
	ZoneService           happydns.ZoneServiceUsecase
}

//	@title			happyDomain API
//	@version		0.1
//	@description	Finally a simple interface for domain names.

//	@contact.name	happyDomain team
//	@contact.email	contact+api@happydomain.org

//	@license.name	GNU Affero General Public License v3.0 or later
//	@license.url	https://spdx.org/licenses/AGPL-3.0-or-later.html

//	@host		localhost:8081
//	@BasePath	/api

//	@securityDefinitions.basic	BasicAuth

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Description for what is this security definition being used

func DeclareRoutes(cfg *happydns.Options, router *gin.RouterGroup, dep Dependencies) {
	baseRoutes := router.Group("")

	declareRouteSwagger(cfg, baseRoutes)

	apiRoutes := router.Group("/api")

	lc := DeclareAuthenticationRoutes(cfg, baseRoutes, apiRoutes, dep.Authentication, dep.CaptchaVerifier, dep.FailureTracker)
	auc := DeclareAuthUserRoutes(apiRoutes, dep.AuthUser, lc)
	DeclareProviderSpecsRoutes(apiRoutes, dep.ProviderSpecs)
	DeclareRegistrationRoutes(apiRoutes, dep.AuthUser, dep.CaptchaVerifier)
	DeclareResolverRoutes(apiRoutes, dep.Resolver)
	DeclareServiceSpecsRoutes(apiRoutes, dep.ServiceSpecs)
	DeclareUserRecoveryRoutes(apiRoutes, dep.AuthUser, auc)
	DeclareVersionRoutes(apiRoutes)

	apiAuthRoutes := router.Group("/api")

	if cfg.NoAuth {
		apiAuthRoutes.Use(middleware.NoAuthMiddleware(dep.Authentication))
	} else {
		apiAuthRoutes.Use(middleware.JwtAuthMiddleware(dep.Authentication, cfg.JWTSigningMethod, cfg.JWTSecretKey))
		apiAuthRoutes.Use(middleware.SessionMiddleware(dep.Authentication))
	}
	apiAuthRoutes.Use(middleware.AuthRequired())

	DeclareAuthenticationCheckRoutes(apiAuthRoutes, lc)
	DeclareDomainRoutes(apiAuthRoutes, dep.Domain, dep.DomainLog, dep.RemoteZoneImporter, dep.ZoneImporter, dep.Zone, dep.ZoneCorrectionApplier, dep.ZoneService, dep.Service)
	DeclareProviderRoutes(apiAuthRoutes, dep.Provider)
	DeclareProviderSettingsRoutes(apiAuthRoutes, dep.ProviderSettings)
	DeclareRecordRoutes(apiAuthRoutes)
	DeclareUsersRoutes(apiAuthRoutes, dep.User, lc)
	DeclareSessionRoutes(apiAuthRoutes, dep.Session)
}
