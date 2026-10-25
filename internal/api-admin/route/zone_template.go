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

	"git.happydns.org/happyDomain/internal/api-admin/controller"
)

// declareZoneTemplateRoutes declares the routes of the global zone
// templates, the same as those of the users. They are listed again rather
// than shared: the documentation of the administrative API lies on the
// handlers of its own controller.
func declareZoneTemplateRoutes(router *gin.RouterGroup, dep Dependencies) {
	if dep.ZoneTemplate == nil {
		return
	}

	zc := controller.NewZoneTemplateController(dep.ZoneTemplate)
	templates := router.Group("/templates")
	templates.GET("", zc.ListZoneTemplates)
	templates.POST("", zc.CreateZoneTemplate)
	templates.POST("/import", zc.ImportZoneTemplate)

	template := templates.Group("/:templateId")
	template.GET("", zc.GetZoneTemplate)
	template.PUT("", zc.UpdateZoneTemplate)
	template.DELETE("", zc.DeleteZoneTemplate)

	template.POST("/records", zc.AddZoneTemplateRecords)
	template.POST("/records/delete", zc.DeleteZoneTemplateRecords)
	template.PATCH("/records", zc.UpdateZoneTemplateRecord)

	template.POST("/:subdomain/services", zc.AddZoneTemplateService)
	template.PUT("/:subdomain/services/:serviceId", zc.UpdateZoneTemplateService)
	template.DELETE("/:subdomain/services/:serviceId", zc.DeleteZoneTemplateService)
}
