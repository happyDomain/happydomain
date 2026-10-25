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
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

// ZoneTemplateController handles the zone templates of the user logged in.
type ZoneTemplateController struct {
	zoneTemplateService happydns.ZoneTemplateUsecase
	domainService       happydns.DomainUsecase
	zoneService         happydns.ZoneUsecase
}

// NewZoneTemplateController builds the controller of the API of the users.
func NewZoneTemplateController(zoneTemplateService happydns.ZoneTemplateUsecase, domainService happydns.DomainUsecase, zoneService happydns.ZoneUsecase) *ZoneTemplateController {
	return &ZoneTemplateController{
		zoneTemplateService: zoneTemplateService,
		domainService:       domainService,
		zoneService:         zoneService,
	}
}

// user is the user the request acts for; it answers the request itself when
// there is none.
func (tc *ZoneTemplateController) user(c *gin.Context) (*happydns.User, bool) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "User not defined."})
		return nil, false
	}
	return user, true
}

// templateId reads the identifier of the template from the path; it answers
// the request itself when it is malformed.
func templateId(c *gin.Context) (happydns.Identifier, bool) {
	id, err := happydns.NewIdentifierFromString(c.Param("templateId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid template identifier."})
		return nil, false
	}
	return id, true
}

// serviceId reads the identifier of the service from the path; it answers
// the request itself when it is malformed.
func serviceId(c *gin.Context) (happydns.Identifier, bool) {
	id, err := happydns.NewIdentifierFromString(c.Param("serviceId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid service identifier."})
		return nil, false
	}
	return id, true
}

// respond answers with the template changed, or the error.
func respond(c *gin.Context, t *happydns.ZoneTemplate, err error) {
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

// ListZoneTemplates lists the templates the user can apply.
//
//	@Summary	List zone templates
//	@Schemes
//	@Description	List the zone templates of the user, then the global ones, offered to every user.
//	@Tags			zone-templates
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.ZoneTemplate
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/templates [get]
func (tc *ZoneTemplateController) ListZoneTemplates(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}

	templates, err := tc.zoneTemplateService.ListZoneTemplates(user)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, templates)
}

// CreateZoneTemplate creates a template, empty or from a zone.
//
//	@Summary	Create a zone template
//	@Schemes
//	@Description	Create an empty zone template, or save a zone of a domain of the user as a template, but for its origin (SOA and NS records). A zone of the domain without zone identifier is its zone being edited.
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			body	body		happydns.ZoneTemplateCreationForm	true	"Template to create"
//	@Success		201		{object}	happydns.ZoneTemplate
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403		{object}	happydns.ErrorResponse	"Too many templates"
//	@Failure		404		{object}	happydns.ErrorResponse	"Domain or zone not found"
//	@Router			/templates [post]
func (tc *ZoneTemplateController) CreateZoneTemplate(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}

	var form happydns.ZoneTemplateCreationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	var t *happydns.ZoneTemplate
	var err error
	if form.DomainId == nil {
		t, err = tc.zoneTemplateService.CreateZoneTemplate(user, form.ZoneTemplateInput)
	} else {
		var domain *happydns.Domain
		var zone *happydns.Zone
		domain, zone, err = tc.zoneOf(user, *form.DomainId, form.ZoneId)
		if err == nil {
			t, err = tc.zoneTemplateService.CreateZoneTemplateFromZone(user, form.ZoneTemplateInput, domain, zone)
		}
	}
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, t)
}

// zoneOf finds a zone of a domain of the user: the one being edited when
// zoneId is nil.
func (tc *ZoneTemplateController) zoneOf(user *happydns.User, domainId happydns.Identifier, zoneId *happydns.Identifier) (*happydns.Domain, *happydns.Zone, error) {
	domain, err := tc.domainService.GetUserDomain(user, domainId)
	if err != nil {
		return nil, nil, err
	}

	id := zoneId
	if id == nil {
		if len(domain.ZoneHistory) == 0 {
			return nil, nil, happydns.ValidationError{Msg: "This domain has no zone yet."}
		}
		id = &domain.ZoneHistory[0]
	}

	zone, err := tc.zoneService.LoadZoneFromId(domain, *id)
	if err != nil {
		return nil, nil, err
	}
	return domain, zone, nil
}

// ImportZoneTemplate makes a template from a zone file.
//
//	@Summary	Import a zone template
//	@Schemes
//	@Description	Make a zone template from a BIND-style zone file, but for its origin (SOA and NS records).
//	@Tags			zone-templates
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			name		formData	string	true	"Name of the template"
//	@Param			description	formData	string	false	"Description of the template"
//	@Param			origin		formData	string	false	"Domain the zone file is written for; example.com. when empty"
//	@Param			zone		formData	file	true	"Zone file to import"
//	@Success		201			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input or unable to read the zone file"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Too many templates"
//	@Router			/templates/import [post]
func (tc *ZoneTemplateController) ImportZoneTemplate(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}

	fd, _, err := c.Request.FormFile("zone")
	if err != nil {
		log.Printf("Error when retrieving the zone file of a template from %s: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Unable to read your zone file: something is wrong in your request"})
		return
	}
	defer fd.Close()

	t, err := tc.zoneTemplateService.ImportZoneTemplate(user, happydns.ZoneTemplateInput{
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
	}, c.PostForm("origin"), fd)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, t)
}

// GetZoneTemplate shows a template.
//
//	@Summary	Get a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string	true	"Template identifier"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid template identifier"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId} [get]
func (tc *ZoneTemplateController) GetZoneTemplate(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}

	t, err := tc.zoneTemplateService.GetZoneTemplate(user, id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, t)
}

// UpdateZoneTemplate changes what describes a template.
//
//	@Summary	Update a zone template
//	@Schemes
//	@Description	Change the name and the description of a template. The global templates are only changed by the administrators.
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string						true	"Template identifier"
//	@Param			body		body		happydns.ZoneTemplateInput	true	"What describes the template"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid template identifier or input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId} [put]
func (tc *ZoneTemplateController) UpdateZoneTemplate(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}

	var input happydns.ZoneTemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	t, err := tc.zoneTemplateService.UpdateZoneTemplate(user, id, input)
	respond(c, t, err)
}

// DeleteZoneTemplate deletes a template.
//
//	@Summary	Delete a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Security		securitydefinitions.basic
//	@Param			templateId	path	string	true	"Template identifier"
//	@Success		204
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid template identifier"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403	{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404	{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId} [delete]
func (tc *ZoneTemplateController) DeleteZoneTemplate(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}

	if err := tc.zoneTemplateService.DeleteZoneTemplate(user, id); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// bindService reads the service of the body; it answers the request itself
// when it is malformed.
func bindService(c *gin.Context) (*happydns.Service, bool) {
	var msg happydns.ServiceMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return nil, false
	}

	svc, err := serviceUC.ParseService(&msg)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return nil, false
	}
	return svc, true
}

// AddZoneTemplateService adds a service to a template.
//
//	@Summary	Add a service to a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string				true	"Template identifier"
//	@Param			subdomain	path		string				true	"Subdomain of the service, relative to the domains the template is applied to (@ for the domain itself)"
//	@Param			body		body		happydns.Service	true	"Service to add"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/{subdomain}/services [post]
func (tc *ZoneTemplateController) AddZoneTemplateService(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}
	svc, ok := bindService(c)
	if !ok {
		return
	}

	t, err := tc.zoneTemplateService.AddZoneTemplateService(user, id, happydns.Subdomain(c.Param("subdomain")), svc)
	respond(c, t, err)
}

// UpdateZoneTemplateService changes a service of a template.
//
//	@Summary	Update a service of a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string				true	"Template identifier"
//	@Param			subdomain	path		string				true	"Subdomain of the service, relative to the domains the template is applied to (@ for the domain itself)"
//	@Param			serviceId	path		string				true	"Service identifier"
//	@Param			body		body		happydns.Service	true	"Service to update"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template or service not found"
//	@Router			/templates/{templateId}/{subdomain}/services/{serviceId} [put]
func (tc *ZoneTemplateController) UpdateZoneTemplateService(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}
	svcId, ok := serviceId(c)
	if !ok {
		return
	}
	svc, ok := bindService(c)
	if !ok {
		return
	}

	t, err := tc.zoneTemplateService.UpdateZoneTemplateService(user, id, happydns.Subdomain(c.Param("subdomain")), svcId, svc)
	respond(c, t, err)
}

// DeleteZoneTemplateService removes a service from a template.
//
//	@Summary	Delete a service of a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string	true	"Template identifier"
//	@Param			subdomain	path		string	true	"Subdomain of the service, relative to the domains the template is applied to (@ for the domain itself)"
//	@Param			serviceId	path		string	true	"Service identifier"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid identifier"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template or service not found"
//	@Router			/templates/{templateId}/{subdomain}/services/{serviceId} [delete]
func (tc *ZoneTemplateController) DeleteZoneTemplateService(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}
	svcId, ok := serviceId(c)
	if !ok {
		return
	}

	t, err := tc.zoneTemplateService.DeleteZoneTemplateService(user, id, happydns.Subdomain(c.Param("subdomain")), svcId)
	respond(c, t, err)
}

// bindRecords reads the records of the body; it answers the request itself
// when it is malformed.
func bindRecords(c *gin.Context) ([]string, bool) {
	var records []string
	if err := c.ShouldBindJSON(&records); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return nil, false
	}
	return records, true
}

// AddZoneTemplateRecords adds records to a template.
//
//	@Summary	Add records to a zone template
//	@Schemes
//	@Description	Add records, given as text relative to example.com., to the services of a template.
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string		true	"Template identifier"
//	@Param			body		body		[]string	true	"Records to add as text, one record per line array"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/records [post]
func (tc *ZoneTemplateController) AddZoneTemplateRecords(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}
	records, ok := bindRecords(c)
	if !ok {
		return
	}

	t, err := tc.zoneTemplateService.AddZoneTemplateRecords(user, id, records)
	respond(c, t, err)
}

// DeleteZoneTemplateRecords removes records from a template.
//
//	@Summary	Delete records of a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string		true	"Template identifier"
//	@Param			body		body		[]string	true	"Records to delete as text, one record per line array"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/records/delete [post]
func (tc *ZoneTemplateController) DeleteZoneTemplateRecords(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}
	records, ok := bindRecords(c)
	if !ok {
		return
	}

	t, err := tc.zoneTemplateService.DeleteZoneTemplateRecords(user, id, records)
	respond(c, t, err)
}

// UpdateZoneTemplateRecord replaces a record of a template.
//
//	@Summary	Update a record of a zone template
//	@Schemes
//	@Tags			zone-templates
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			templateId	path		string						true	"Template identifier"
//	@Param			body		body		happydns.UpdateRecordForm	true	"Record to update as text"
//	@Success		200			{object}	happydns.ZoneTemplate
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403			{object}	happydns.ErrorResponse	"Global template"
//	@Failure		404			{object}	happydns.ErrorResponse	"Template not found"
//	@Router			/templates/{templateId}/records [patch]
func (tc *ZoneTemplateController) UpdateZoneTemplateRecord(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	id, ok := templateId(c)
	if !ok {
		return
	}

	var form happydns.UpdateRecordForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	t, err := tc.zoneTemplateService.UpdateZoneTemplateRecord(user, id, form)
	respond(c, t, err)
}

// ApplyZoneTemplate applies a template to a zone.
//
//	@Summary	Apply a zone template
//	@Schemes
//	@Description	Apply a template to a zone, merging its services with those of the zone or overwriting them. The origin of the zone (SOA and NS records) is kept. A published zone is not changed: an editable zone is derived from it first.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string							true	"Domain identifier"
//	@Param			zoneId		path		string							true	"Zone identifier"
//	@Param			body		body		happydns.ApplyZoneTemplateForm	true	"Template to apply, and how"
//	@Success		200			{object}	happydns.Zone	"The zone changed"
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain, zone or template not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/apply_template [post]
func (tc *ZoneTemplateController) ApplyZoneTemplate(c *gin.Context) {
	user, ok := tc.user(c)
	if !ok {
		return
	}
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var form happydns.ApplyZoneTemplateForm
	if err := c.ShouldBindJSON(&form); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	zone, err := tc.zoneTemplateService.ApplyZoneTemplate(user, domain, zone, form)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}
