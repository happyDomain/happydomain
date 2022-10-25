// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
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

	dnscontrol "github.com/StackExchange/dnscontrol/v3/providers"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/providers"
	"git.happydns.org/happydomain/storage"
)

func declareProvidersRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.GET("/providers", getProviders)
	router.POST("/providers", addProvider)

	apiProvidersMetaRoutes := router.Group("/providers/:pid")
	apiProvidersMetaRoutes.Use(ProviderMetaHandler)

	apiProvidersMetaRoutes.DELETE("", deleteProvider)

	apiProviderRoutes := router.Group("/providers/:pid")
	apiProviderRoutes.Use(ProviderHandler)

	apiProviderRoutes.GET("", GetProvider)
	apiProviderRoutes.PUT("", UpdateProvider)

	apiProviderRoutes.GET("/domains", getDomainsHostedByProvider)
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
		log.Printf("%s: unable to find provider %s: %s", c.ClientIP(), ust.Type, err.Error())
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

func GetProvider(c *gin.Context) {
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
		log.Println("%s unable to CreateProvider: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create the given provider. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, s)
}

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
		log.Println("%s unable to UpdateProvider: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update the provider. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, src)
}

func deleteProvider(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	providermeta := c.MustGet("providermeta").(*happydns.ProviderMeta)

	// Check if the provider has no more domain associated
	domains, err := storage.MainStore.GetDomains(user)
	if err != nil {
		log.Println("%s unable to GetDomains for user id=%x email=%s: %s", c.ClientIP(), user.Id, user.Email, err.Error())
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
		log.Println("%s unable to DeleteProvider %x for user id=%x email=%s: %s", c.ClientIP(), providermeta.Id, user.Id, user.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to delete your provider. Please try again later."})
		return
	}

	c.JSON(http.StatusNoContent, nil)
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
