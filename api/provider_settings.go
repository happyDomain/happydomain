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
	"git.happydns.org/happydns/providers"
	"git.happydns.org/happydns/storage"
)

func declareProviderSettingsRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.POST("/providers/_specs/:ssid/settings", func(c *gin.Context) {
		getProviderSettingsState(cfg, c)
	})
	//router.POST("/domains/:domain/zone/:zoneid/:subdomain/provider_settings/:psid", getProviderSettingsState)
}

type ProviderSettingsState struct {
	FormState
	happydns.Provider
}

type ProviderSettingsResponse struct {
	FormResponse
	happydns.Provider `json:"Provider,omitempty"`
}

func getProviderSettingsState(cfg *config.Options, c *gin.Context) {
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	ssid := string(c.Param("ssid"))

	src, err := providers.FindProvider(ssid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find your provider: %s", err.Error())})
		return
	}

	var uss ProviderSettingsState
	uss.Provider = src
	err = c.ShouldBindJSON(&uss)
	if err != nil {
		log.Printf("%s sends invalid ProviderSettingsState JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return

	}

	form, err := formDoState(cfg, c, &uss.FormState, src, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != forms.DoneForm {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		} else if _, err = src.NewDNSServiceProvider(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		} else if uss.Id == nil {
			// Create a new Provider
			s, err := storage.MainStore.CreateProvider(user, src, uss.Name)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
				return
			}

			c.JSON(http.StatusOK, ProviderSettingsResponse{
				Provider:     s,
				FormResponse: FormResponse{Redirect: uss.Redirect},
			})
			return
		} else {
			// Update an existing Provider
			s, err := storage.MainStore.GetProvider(user, int64(uss.Id.(float64)))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
				return
			}
			s.Comment = uss.Name
			s.Provider = uss.Provider

			err = storage.MainStore.UpdateProvider(s)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
				return
			}

			c.JSON(http.StatusOK, ProviderSettingsResponse{
				Provider:     s,
				FormResponse: FormResponse{Redirect: uss.Redirect},
			})
			return
		}
	}

	c.JSON(http.StatusOK, ProviderSettingsResponse{
		FormResponse: FormResponse{From: form},
	})
}
