// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/forms"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/storage"
)

func declareServiceSettingsRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.POST("/services/*psid", func(c *gin.Context) {
		getServiceSettingsState(cfg, c)
	})
}

type ServiceSettingsState struct {
	FormState
	happydns.Service
}

type ServiceSettingsResponse struct {
	FormResponse
	Services map[string][]*happydns.ServiceCombined `json:"services,omitempty"`
}

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

	form, err := formDoState(cfg, c, &ups.FormState, ups.Service, forms.GenDefaultSettingsForm)

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
			err = zone.EraseServiceWithoutMeta(subdomain, domain.DomainName, ups.Id.([]byte), ups)
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
			Services:     zone.Services,
			FormResponse: FormResponse{Redirect: ups.Redirect},
		})
		return
	}

	c.JSON(http.StatusOK, ProviderSettingsResponse{
		FormResponse: FormResponse{From: form},
	})
}
