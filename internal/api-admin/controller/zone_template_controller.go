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

package controller

import (
	"github.com/gin-gonic/gin"

	apicontroller "git.happydns.org/happyDomain/internal/api/controller"
	happydns "git.happydns.org/happyDomain/model"
)

// ZoneTemplateController manages the global zone templates, offered to every
// user. It acts as the controller of the API of the users does, without user.
type ZoneTemplateController struct {
	tc *apicontroller.ZoneTemplateController
}

func NewZoneTemplateController(zoneTemplateService happydns.ZoneTemplateUsecase) *ZoneTemplateController {
	return &ZoneTemplateController{
		tc: apicontroller.NewAdminZoneTemplateController(zoneTemplateService),
	}
}

// ListZoneTemplates is the administrative side of the route of the same name.
//
//	@Summary	List the global zone templates
//	@Schemes
//	@Description	List the zone templates offered to every user.
//	@Tags			admin-zone-templates
//	@Produce		json
//	@Success		200		{array}		happydns.ZoneTemplate
//	@Router			/templates [get]
func (zc *ZoneTemplateController) ListZoneTemplates(c *gin.Context) {
	zc.tc.ListZoneTemplates(c)
}

// CreateZoneTemplate is the administrative side of the route of the same name.
//
//	@Summary	Create a global zone template
//	@Schemes
//	@Description	Create an empty zone template offered to every user.
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.ZoneTemplateInput	true	"Template to create"
//	@Success		201		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Router			/templates [post]
func (zc *ZoneTemplateController) CreateZoneTemplate(c *gin.Context) {
	zc.tc.CreateZoneTemplate(c)
}

// ImportZoneTemplate is the administrative side of the route of the same name.
//
//	@Summary	Import a global zone template
//	@Schemes
//	@Description	Make a zone template offered to every user from a BIND-style zone file, but for its origin (SOA and NS records).
//	@Tags			admin-zone-templates
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			name		formData	string	true	"Name of the template"
//	@Param			description	formData	string	false	"Description of the template"
//	@Param			origin		formData	string	false	"Domain the zone file is written for; example.com. when empty"
//	@Param			zone		formData	file	true	"Zone file to import"
//	@Success		201		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Router			/templates/import [post]
func (zc *ZoneTemplateController) ImportZoneTemplate(c *gin.Context) {
	zc.tc.ImportZoneTemplate(c)
}

// GetZoneTemplate is the administrative side of the route of the same name.
//
//	@Summary	Get a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Produce		json
//	@Param			templateId	path		string	true	"Template identifier"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId} [get]
func (zc *ZoneTemplateController) GetZoneTemplate(c *gin.Context) {
	zc.tc.GetZoneTemplate(c)
}

// UpdateZoneTemplate is the administrative side of the route of the same name.
//
//	@Summary	Update a global zone template
//	@Schemes
//	@Description	Change the name and the description of a global template.
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			templateId	path		string	true	"Template identifier"
//	@Param			body		body		happydns.ZoneTemplateInput	true	"What describes the template"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId} [put]
func (zc *ZoneTemplateController) UpdateZoneTemplate(c *gin.Context) {
	zc.tc.UpdateZoneTemplate(c)
}

// DeleteZoneTemplate is the administrative side of the route of the same name.
//
//	@Summary	Delete a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Param			templateId	path	string	true	"Template identifier"
//	@Success		204
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId} [delete]
func (zc *ZoneTemplateController) DeleteZoneTemplate(c *gin.Context) {
	zc.tc.DeleteZoneTemplate(c)
}

// AddZoneTemplateRecords is the administrative side of the route of the same name.
//
//	@Summary	Add records to a global zone template
//	@Schemes
//	@Description	Add records, given as text relative to example.com., to the services of a global template.
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			templateId	path		string		true	"Template identifier"
//	@Param			body		body		[]string	true	"Records to add as text, one record per line array"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/records [post]
func (zc *ZoneTemplateController) AddZoneTemplateRecords(c *gin.Context) {
	zc.tc.AddZoneTemplateRecords(c)
}

// DeleteZoneTemplateRecords is the administrative side of the route of the same name.
//
//	@Summary	Delete records of a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			templateId	path		string		true	"Template identifier"
//	@Param			body		body		[]string	true	"Records to delete as text, one record per line array"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/records/delete [post]
func (zc *ZoneTemplateController) DeleteZoneTemplateRecords(c *gin.Context) {
	zc.tc.DeleteZoneTemplateRecords(c)
}

// UpdateZoneTemplateRecord is the administrative side of the route of the same name.
//
//	@Summary	Update a record of a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			templateId	path		string						true	"Template identifier"
//	@Param			body		body		happydns.UpdateRecordForm	true	"Record to update as text"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/records [patch]
func (zc *ZoneTemplateController) UpdateZoneTemplateRecord(c *gin.Context) {
	zc.tc.UpdateZoneTemplateRecord(c)
}

// AddZoneTemplateService is the administrative side of the route of the same name.
//
//	@Summary	Add a service to a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			templateId	path		string				true	"Template identifier"
//	@Param			subdomain	path		string				true	"Subdomain of the service, relative to the domains the template is applied to (@ for the domain itself)"
//	@Param			body		body		happydns.Service	true	"Service to add"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/{subdomain}/services [post]
func (zc *ZoneTemplateController) AddZoneTemplateService(c *gin.Context) {
	zc.tc.AddZoneTemplateService(c)
}

// UpdateZoneTemplateService is the administrative side of the route of the same name.
//
//	@Summary	Update a service of a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Accept			json
//	@Produce		json
//	@Param			templateId	path		string				true	"Template identifier"
//	@Param			subdomain	path		string				true	"Subdomain of the service, relative to the domains the template is applied to (@ for the domain itself)"
//	@Param			serviceId	path		string				true	"Service identifier"
//	@Param			body		body		happydns.Service	true	"Service to update"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/{subdomain}/services/{serviceId} [put]
func (zc *ZoneTemplateController) UpdateZoneTemplateService(c *gin.Context) {
	zc.tc.UpdateZoneTemplateService(c)
}

// DeleteZoneTemplateService is the administrative side of the route of the same name.
//
//	@Summary	Delete a service of a global zone template
//	@Schemes
//	@Tags			admin-zone-templates
//	@Produce		json
//	@Param			templateId	path		string	true	"Template identifier"
//	@Param			subdomain	path		string	true	"Subdomain of the service, relative to the domains the template is applied to (@ for the domain itself)"
//	@Param			serviceId	path		string	true	"Service identifier"
//	@Success		200		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/{subdomain}/services/{serviceId} [delete]
func (zc *ZoneTemplateController) DeleteZoneTemplateService(c *gin.Context) {
	zc.tc.DeleteZoneTemplateService(c)
}
