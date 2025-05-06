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

	"git.happydns.org/happyDomain/api/controller"
	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/model"
)

func DeclareZoneRoutes(router *gin.RouterGroup, dependancies happydns.UsecaseDependancies) {
	zc := controller.NewZoneController(dependancies.GetZoneUsecase(), dependancies.GetDomainUsecase())

	apiZonesRoutes := router.Group("/zone/:zoneid")
	apiZonesRoutes.Use(middleware.ZoneHandler(dependancies.GetZoneUsecase()))

	apiZonesRoutes.GET("", zc.GetZone)

	apiZonesRoutes.POST("/diff/:oldzoneid", zc.DiffZones)
	apiZonesRoutes.POST("/view", zc.ExportZone)
	apiZonesRoutes.POST("/apply_changes", zc.ApplyZoneCorrections)

	apiZonesSubdomainRoutes := apiZonesRoutes.Group("/:subdomain")
	apiZonesSubdomainRoutes.Use(middleware.SubdomainHandler)
	apiZonesSubdomainRoutes.GET("", zc.GetZoneSubdomain)

	DeclareZoneServiceRoutes(apiZonesRoutes, apiZonesSubdomainRoutes, zc, dependancies)
}
