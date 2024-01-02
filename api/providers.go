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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	dnscontrol "github.com/StackExchange/dnscontrol/v4/providers"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
	"git.happydns.org/happyDomain/storage"
)

func declareProvidersRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.GET("/providers", getProviders)
	router.POST("/providers", func(c *gin.Context) {
		if cfg.DisableProviders {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Cannot add provider as DisableProviders parameter is set."})
			return
		}

		addProvider(c)
	})

	apiProvidersMetaRoutes := router.Group("/providers/:pid")
	apiProvidersMetaRoutes.Use(ProviderMetaHandler)

	apiProvidersMetaRoutes.DELETE("", func(c *gin.Context) {
		if cfg.DisableProviders {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Cannot delete provider as DisableProviders parameter is set."})
			return
		}

		deleteProvider(c)
	})

	apiProviderRoutes := router.Group("/providers/:pid")
	apiProviderRoutes.Use(ProviderHandler)

	apiProviderRoutes.GET("", GetProvider)
	apiProviderRoutes.PUT("", func(c *gin.Context) {
		if cfg.DisableProviders {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Cannot update provider as DisableProviders parameter is set."})
			return
		}

		UpdateProvider(c)
	})

	apiProviderRoutes.GET("/domains", getDomainsHostedByProvider)
}

// getDomains retrieves all providers belonging to the user.
//
//	@Summary	Retrieve user's providers
//	@Schemes
//	@Description	Retrieve all DNS providers belonging to the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.Provider
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Unable to retrieve user's domains"
//	@Router			/providers [get]
func getProviders(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	if providers, err := storage.MainStore.GetProviderMetas(user); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	} else if len(providers) > 0 {
		c.JSON(http.StatusOK, providers)
	} else {
		c.JSON(http.StatusOK, []happydns.Provider{})
	}
}

func DecodeProvider(c *gin.Context) (*happydns.ProviderCombined, int, error) {
	buf, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Unable to read input: %w", err)
	}

	var ust happydns.ProviderMeta
	err = json.Unmarshal(buf, &ust)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Unable to parse input as ProviderMeta: %w", err)
	}

	us, err := providers.FindProvider(ust.Type)
	if err != nil {
		log.Printf("%s: unable to find provider %s: %s", c.ClientIP(), ust.Type, err.Error())
		return nil, http.StatusInternalServerError, fmt.Errorf("Sorry, we were unable to find the kind of provider in our database. Please report this issue.")
	}

	src := &happydns.ProviderCombined{
		Provider:     us,
		ProviderMeta: ust,
	}

	err = json.Unmarshal(buf, &src)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Unable to parse input as Provider: %w", err)
	}

	err = src.Validate()
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Unable to validate input: %w", err)
	}

	return src, http.StatusOK, nil
}

func ProviderMetaHandler(c *gin.Context) {
	// Extract provider ID
	pid, err := happydns.NewIdentifierFromString(string(c.Param("pid")))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid provider id: %s", err.Error())})
		return
	}

	// Get a valid user
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
		return
	}

	// Retrieve provider meta
	providermeta, err := storage.MainStore.GetProviderMeta(user, pid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Provider not found."})
		return
	}

	// Continue
	c.Set("providermeta", providermeta)

	c.Next()
}

func ProviderHandler(c *gin.Context) {
	// Extract provider ID
	pid, err := happydns.NewIdentifierFromString(string(c.Param("pid")))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid provider id: %s", err.Error())})
		return
	}

	// Get a valid user
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
		return
	}

	// Retrieve provider
	provider, err := storage.MainStore.GetProvider(user, pid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Provider not found."})
		return
	}

	// Continue
	c.Set("provider", provider)
	c.Set("providermeta", provider.ProviderMeta)

	c.Next()
}

// GetProvider retrieves information about a given Provider owned by the user.
//
//	@Summary	Retrieve Provider information.
//	@Schemes
//	@Description	Retrieve information in the database about a given Provider owned by the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string	true	"Provider identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.ProviderCombined
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Provider not found"
//	@Router			/providers/{providerId} [get]
func GetProvider(c *gin.Context) {
	provider := c.MustGet("provider").(*happydns.ProviderCombined)

	c.JSON(http.StatusOK, provider)
}

// addProvider appends a new provider.
//
//	@Summary	Add a new provider
//	@Schemes
//	@Description	Append a new provider for the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			body	body	happydns.ProviderMinimal	true	"Provider to add"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider
//	@Failure		400	{object}	happydns.Error	"Error in received data"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		500	{object}	happydns.Error	"Unable to retrieve current user's providers"
//	@Router			/providers [post]
func addProvider(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	src, statuscode, err := DecodeProvider(c)
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	s, err := storage.MainStore.CreateProvider(user, src.Provider, src.Comment)
	if err != nil {
		log.Printf("%s unable to CreateProvider: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create the given provider. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, s)
}

// UpdateProvider updates the information about a given Provider owned by the user.
//
//	@Summary	Update Provider information.
//	@Schemes
//	@Description	Updates the information about a given Provider owned by the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string				true	"Provider identifier"
//	@Param			body		body	happydns.Provider	true	"The new object overriding the current provider"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider
//	@Failure		400	{object}	happydns.Error	"Invalid input"
//	@Failure		400	{object}	happydns.Error	"Identifier changed"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Provider not found"
//	@Failure		500	{object}	happydns.Error	"Database writing error"
//	@Router			/providers/{providerId} [put]
func UpdateProvider(c *gin.Context) {
	provider := c.MustGet("provider").(*happydns.ProviderCombined)

	src, statuscode, err := DecodeProvider(c)
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	src.Id = provider.Id
	src.OwnerId = provider.OwnerId

	if err := storage.MainStore.UpdateProvider(src); err != nil {
		log.Printf("%s unable to UpdateProvider: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update the provider. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, src)
}

// deleteProvider removes a provider from the database.
//
//	@Summary	Delete a Provider.
//	@Schemes
//	@Description	Delete a Provider from the database. It is required that no Domain are still managed by this Provider before calling this route.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string	true	"Provider identifier"
//	@Security		securitydefinitions.basic
//	@Success		204	"Provider deleted"
//	@Failure		400	{object}	happydns.Error	"Invalid input"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Provider not found"
//	@Failure		500	{object}	happydns.Error	"Database writing error"
//	@Router			/providers/{providerId} [delete]
func deleteProvider(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	providermeta := c.MustGet("providermeta").(*happydns.ProviderMeta)

	// Check if the provider has no more domain associated
	domains, err := storage.MainStore.GetDomains(user)
	if err != nil {
		log.Printf("%s unable to GetDomains for user id=%x email=%s: %s", c.ClientIP(), user.Id, user.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to perform this action. Please try again later."})
		return
	}

	for _, domain := range domains {
		if domain.IdProvider.Equals(providermeta.Id) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "You cannot delete this provider because there is still some domains associated with it."})
			return
		}
	}

	if err := storage.MainStore.DeleteProvider(providermeta); err != nil {
		log.Printf("%s unable to DeleteProvider %x for user id=%x email=%s: %s", c.ClientIP(), providermeta.Id, user.Id, user.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to delete your provider. Please try again later."})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// getDomainsHostedByProvider lists domains available to management from the given Provider.
//
//	@Summary	Lists manageable domains from the Provider.
//	@Schemes
//	@Description	List domains available from the given Provider.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string	true	"Provider identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.ProviderCombined
//	@Failure		400	{object}	happydns.Error	"Unable to instantiate the provider"
//	@Failure		400	{object}	happydns.Error	"The provider doesn't support domain listing"
//	@Failure		400	{object}	happydns.Error	"Provider error"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Provider not found"
//	@Router			/providers/{providerId}/domains [get]
func getDomainsHostedByProvider(c *gin.Context) {
	provider := c.MustGet("provider").(*happydns.ProviderCombined)

	p, err := provider.NewDNSServiceProvider()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to instantiate the provider: %s", err.Error())})
		return
	}

	sr, ok := p.(dnscontrol.ZoneLister)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Provider doesn't support domain listing."})
		return
	}

	domains, err := sr.ListZones()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domains)
}
