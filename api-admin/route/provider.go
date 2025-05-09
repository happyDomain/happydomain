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

	"git.happydns.org/happyDomain/api-admin/controller"
	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

func declareProviderRoutes(router *gin.RouterGroup, dependancies happydns.UsecaseDependancies, store storage.Storage) {
	pc := controller.NewProviderController(dependancies.GetProviderService(false), store)

	router.GET("/providers", pc.ListProviders)
	router.POST("/providers", pc.AddProvider)
	router.DELETE("/providers", pc.ClearProviders)

	apiProvidersMetaRoutes := router.Group("/providers/:pid")
	apiProvidersMetaRoutes.Use(middleware.ProviderMetaHandler(dependancies.GetProviderService(false)))

	apiProvidersMetaRoutes.PUT("", pc.UpdateProvider)
	apiProvidersMetaRoutes.DELETE("", pc.DeleteProvider)

	apiProvidersRoutes := router.Group("/providers/:pid")
	apiProvidersRoutes.Use(middleware.ProviderHandler(dependancies.GetProviderService(false)))

	apiProvidersRoutes.GET("", pc.GetProvider)

	declareDomainRoutes(apiProvidersRoutes, dependancies, store)
}
