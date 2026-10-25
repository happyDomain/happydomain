// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// DeclareZoneTemplateRoutes declares the routes of the zone templates, and
// the one applying a template to a zone of a domain.
func DeclareZoneTemplateRoutes(apiAuthRoutes *gin.RouterGroup, zoneTemplateUC happydns.ZoneTemplateUsecase, domainUC happydns.DomainUsecase, zoneUC happydns.ZoneUsecase) {
	tc := controller.NewZoneTemplateController(zoneTemplateUC, domainUC, zoneUC)

	templates := apiAuthRoutes.Group("/templates")
	templates.GET("", tc.ListZoneTemplates)
	templates.POST("", tc.CreateZoneTemplate)
	templates.POST("/import", tc.ImportZoneTemplate)

	template := templates.Group("/:templateId")
	template.GET("", tc.GetZoneTemplate)
	template.PUT("", tc.UpdateZoneTemplate)
	template.DELETE("", tc.DeleteZoneTemplate)

	template.POST("/records", tc.AddZoneTemplateRecords)
	template.POST("/records/delete", tc.DeleteZoneTemplateRecords)
	template.PATCH("/records", tc.UpdateZoneTemplateRecord)

	template.POST("/:subdomain/services", tc.AddZoneTemplateService)
	template.PUT("/:subdomain/services/:serviceId", tc.UpdateZoneTemplateService)
	template.DELETE("/:subdomain/services/:serviceId", tc.DeleteZoneTemplateService)

	apply := apiAuthRoutes.Group("/domains/:domain/zone/:zoneid")
	apply.Use(middleware.DomainHandler(domainUC, false), middleware.ZoneHandler(zoneUC))
	apply.POST("/apply_template", tc.ApplyZoneTemplate)
}
