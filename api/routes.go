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

package api

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
)

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

func DeclareRoutes(cfg *config.Options, router *gin.Engine) {
	authRoutes := router.Group("")
	authRoutes.Use(authMiddleware(cfg, false))
	declareApiCompatRoutes(cfg, authRoutes)

	apiRoutes := router.Group("/api")

	declareAuthenticationRoutes(cfg, apiRoutes)
	declareProviderSpecsRoutes(apiRoutes)
	declareResolverRoutes(apiRoutes)
	declareServiceSpecsRoutes(apiRoutes)
	declareUsersRoutes(cfg, apiRoutes)
	DeclareVersionRoutes(apiRoutes)

	apiAuthRoutes := router.Group("/api")
	apiAuthRoutes.Use(authMiddleware(cfg, false))

	declareDomainsRoutes(cfg, apiAuthRoutes)
	declareProvidersRoutes(cfg, apiAuthRoutes)
	declareProviderSettingsRoutes(cfg, apiAuthRoutes)
	declareUsersAuthRoutes(cfg, apiAuthRoutes)

	declareRouteSwagger(cfg, router)
}
