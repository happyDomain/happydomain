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

	"git.happydns.org/happyDomain/internal/api-admin/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/storage"
)

func declareDomainRoutes(router *gin.RouterGroup, dep Dependencies, store storage.Storage) {
	dc := controller.NewDomainController(
		dep.Domain,
		dep.RemoteZoneImporter,
		dep.ZoneImporter,
		store,
	)

	router.GET("/domains", dc.ListDomains)
	router.POST("/domains", dc.NewDomain)
	router.DELETE("/domains", dc.ClearDomains)

	router.DELETE("/domains/:domain", dc.DeleteDomain)

	apiDomainsRoutes := router.Group("/domains/:domain")
	apiDomainsRoutes.Use(middleware.DomainHandler(dep.Domain, true))

	apiDomainsRoutes.GET("", dc.GetDomain)
	apiDomainsRoutes.PUT("", dc.UpdateDomain)

	apiDomainsRoutes.PUT("/zones", dc.UpdateZones)
	declareZoneRoutes(apiDomainsRoutes, dep, store)
}
