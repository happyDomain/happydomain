// Copyright or © or Copr. happyDNS (2021)
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
	"strconv"

	dnscontrol "github.com/StackExchange/dnscontrol/v3/providers"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/providers"
	"git.happydns.org/happydns/storage"
)

func declareProvidersRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.GET("/providers", getProviders)
	router.POST("/providers", addProvider)

	apiProviderRoutes := router.Group("/providers/:pid")
	apiProviderRoutes.Use(ProviderHandler)

	apiProviderRoutes.GET("", getProvider)
	//router.PUT("/api/providers/:sid", apiAuthHandler(providerHandler(updateProvider)))
	//router.DELETE("/api/providers/:sid", apiAuthHandler(providerMetaHandler(deleteProvider)))

	apiProviderRoutes.GET("/domains", getDomainsHostedByProvider)

	//router.GET("/api/providers/:sid/domains_with_actions", apiAuthHandler(providerHandler(getDomainsWithActionsHostedByProvider)))
	//router.POST("/api/providers/:sid/domains_with_actions", apiAuthHandler(providerHandler(doDomainsWithActionsHostedByProvider)))

	//router.GET("/api/providers/:sid/available_resource_types", apiAuthHandler(providerHandler(getAvailableResourceTypes)))
}

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
	var ust happydns.ProviderMeta
	err := c.ShouldBindJSON(&ust)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	us, err := providers.FindProvider(ust.Type)
	if err != nil {
		log.Printf("%s: unable to find provider %s: %w", c.ClientIP(), ust.Type, err)
		return nil, http.StatusInternalServerError, fmt.Errorf("Sorry, we were unable to find the kind of provider in our database. Please report this issue.")
	}

	src := &happydns.ProviderCombined{
		us,
		ust,
	}

	err = c.ShouldBindJSON(&src)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	err = src.Validate()
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	return src, http.StatusOK, nil
}

func ProviderHandler(c *gin.Context) {
	// Extract provider ID
	pid, err := strconv.ParseInt(string(c.Param("pid")), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid provider id: %w", err)})
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

func getProvider(c *gin.Context) {
	provider := c.MustGet("provider").(*happydns.ProviderCombined)

	c.JSON(http.StatusOK, provider)
}

func addProvider(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	src, statuscode, err := DecodeProvider(c)
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	s, err := storage.MainStore.CreateProvider(user, src.Provider, src.Comment)
	if err != nil {
		log.Println("%s unable to CreateProvider: %w", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create the given provider. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, s)
}

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
