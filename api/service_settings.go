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
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/forms"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/storage"
)

func declareServiceSettingsRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.POST("/services/*psid", func(c *gin.Context) {
		getServiceSettingsState(cfg, c)
	})
}

type ServiceSettingsState struct {
	FormState
	happydns.Service `json:"Service" swaggertype:"object"`
}

type ServiceSettingsResponse struct {
	Services map[string][]*happydns.ServiceCombined `json:"services,omitempty"`
	Values   map[string]interface{}                 `json:"values,omitempty"`
	Form     *forms.CustomForm                      `json:"form,omitempty"`
}

// getServiceSettingsState creates or updates a Service with human fillable forms.
//
//	@Summary	Assistant to Service creation.
//	@Schemes
//	@Description	This creates or updates a Service with human fillable forms.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		json
//	@Param			serviceType	path	string					true	"The service's type"
//	@Param			body		body	ServiceSettingsState	true	"The current state of the Service's parameters, possibly empty (but not null)"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	ServiceSettingsResponse		"The settings need more rafinement"
//	@Success		200	{object}	happydns.ServiceCombined	"The Service has been created with the given settings"
//	@Failure		400	{object}	happydns.Error				"Invalid input"
//	@Failure		401	{object}	happydns.Error				"Authentication failure"
//	@Failure		404	{object}	happydns.Error				"Service not found"
//	@Router			/service/{serviceType} [post]
func getServiceSettingsState(cfg *config.Options, c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)
	subdomain := c.MustGet("subdomain").(string)

	psid := string(c.Param("psid"))
	// Remove the leading slash
	if len(psid) > 1 {
		psid = psid[1:]
	}

	pvr, err := svcs.FindService(psid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, fmt.Sprintf("Unable to find this service: %s", err.Error()))
		return
	}

	var ups ServiceSettingsState
	ups.Service = pvr
	err = c.ShouldBindJSON(&ups)
	if err != nil {
		log.Printf("%s sends invalid ServiceSettingsState JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	form, p, err := formDoState(cfg, c, &ups.FormState, ups.Service, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != forms.DoneForm {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		} else if ups.Id == nil {
			// Append a new Service
			err = zone.AppendService(subdomain, domain.DomainName, &happydns.ServiceCombined{Service: ups.Service})
			return
		} else {
			// Update an existing Service
			err = zone.EraseServiceWithoutMeta(subdomain, domain.DomainName, *ups.Id, ups)
		}

		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		}

		err = storage.MainStore.UpdateZone(zone)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ServiceSettingsResponse{
			Services: zone.Services,
		})
		return
	}

	c.JSON(http.StatusOK, ServiceSettingsResponse{
		Form:   form,
		Values: p,
	})
}
