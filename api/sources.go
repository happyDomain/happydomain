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
	"strconv"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
	"git.happydns.org/happydns/storage"
)

func declareSourcesRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.GET("/sources", getSources)
	router.POST("/sources", addSource)

	apiSourcesMetaRoutes := router.Group("/sources/:sid")
	apiSourcesMetaRoutes.Use(SourceMetaHandler)

	apiSourcesMetaRoutes.DELETE("", deleteSource)

	apiSourcesRoutes := router.Group("/sources/:sid")
	apiSourcesRoutes.Use(SourceHandler)

	apiSourcesRoutes.GET("", GetSource)
	apiSourcesRoutes.PUT("", UpdateSource)

	apiSourcesRoutes.GET("/domains", getDomainsHostedBySource)

	apiSourcesRoutes.GET("/domains_with_actions", getDomainsWithActionsHostedBySource)
	apiSourcesRoutes.POST("/domains_with_actions", doDomainsWithActionsHostedBySource)

	apiSourcesRoutes.GET("/available_resource_types", getAvailableResourceTypes)
}

func getSources(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	sources, err := storage.MainStore.GetSourceMetas(user)
	if err != nil {
		log.Println("%s unable to GetSourceMetas(%s): %w", c.ClientIP(), user.Email, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to list your sources. Please try again later."})
		return
	}

	if len(sources) == 0 {
		c.JSON(http.StatusNoContent, []happydns.Source{})
	}

	c.JSON(http.StatusOK, sources)

}

func SourceMetaHandler(c *gin.Context) {
	// Extract source ID
	sid, err := strconv.ParseInt(string(c.Param("sid")), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid source id: %w", err)})
		return
	}

	// Get a valid user
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
		return
	}

	// Retrieve source meta
	sourcemeta, err := storage.MainStore.GetSourceMeta(user, sid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Source not found."})
		return
	}

	// Continue
	c.Set("sourcemeta", sourcemeta)

	c.Next()
}

func SourceHandler(c *gin.Context) {
	// Extract source ID
	sid, err := strconv.ParseInt(string(c.Param("sid")), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid source id: %w", err)})
		return
	}

	// Get a valid user
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
		return
	}

	// Retrieve source
	source, err := storage.MainStore.GetSource(user, sid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Source not found."})
		return
	}

	// Continue
	c.Set("source", source)
	c.Set("sourcemeta", source.SourceMeta)

	c.Next()
}

func GetSource(c *gin.Context) {
	source := c.MustGet("source").(*happydns.SourceCombined)

	c.JSON(http.StatusOK, source)
}

func DecodeSource(c *gin.Context) (*happydns.SourceCombined, int, error) {
	var ust happydns.SourceMeta
	err := c.ShouldBindJSON(&ust)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	us, err := sources.FindSource(ust.Type)
	if err != nil {
		log.Printf("%s: unable to find source %s: %w", c.ClientIP(), ust.Type, err)
		return nil, http.StatusInternalServerError, fmt.Errorf("Sorry, we were unable to find the kind of source in our database. Please report this issue.")
	}

	src := &happydns.SourceCombined{
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

func addSource(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	src, statuscode, err := DecodeSource(c)
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	s, err := storage.MainStore.CreateSource(user, src.Source, src.Comment)
	if err != nil {
		log.Println("%s unable to CreateSource: %w", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create the given source. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, s)
}

func UpdateSource(c *gin.Context) {
	source := c.MustGet("source").(*happydns.SourceCombined)

	src, statuscode, err := DecodeSource(c)
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	src.Id = source.Id
	src.OwnerId = source.OwnerId

	if err := storage.MainStore.UpdateSource(src); err != nil {
		log.Println("%s unable to UpdateSource: %w", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update the source. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, src)
}

func deleteSource(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	sourcemeta := c.MustGet("sourcemeta").(*happydns.SourceMeta)

	// Check if the source has no more domain associated
	domains, err := storage.MainStore.GetDomains(user)
	if err != nil {
		log.Println("%s unable to GetDomains for user id=%x email=%s: %w", c.ClientIP(), user.Id, user.Email, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to perform this action. Please try again later."})
		return
	}

	for _, domain := range domains {
		if domain.IdSource == sourcemeta.Id {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "You cannot delete this source because there is still some domains associated with it."})
			return
		}
	}

	if err := storage.MainStore.DeleteSource(sourcemeta); err != nil {
		log.Println("%s unable to DeleteSource %x for user id=%x email=%s: %w", c.ClientIP(), sourcemeta.Id, user.Id, user.Email, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to delete your source. Please try again later."})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func getDomainsHostedBySource(c *gin.Context) {
	source := c.MustGet("source").(*happydns.SourceCombined)

	sr, ok := source.Source.(sources.ListDomainsSource)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Source doesn't support domain listing."})
		return
	}

	domains, err := sr.ListDomains()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domains)
}

func getDomainsWithActionsHostedBySource(c *gin.Context) {
	source := c.MustGet("source").(*happydns.SourceCombined)

	sr, ok := source.Source.(sources.ListDomainsWithActionsSource)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Source doesn't support domain listing."})
		return
	}

	domains, err := sr.ListDomainsWithActions()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domains)
}

func doDomainsWithActionsHostedBySource(c *gin.Context) {
	source := c.MustGet("source").(*happydns.SourceCombined)

	sr, ok := source.Source.(sources.ListDomainsWithActionsSource)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Source doesn't support domain listing."})
		return
	}

	var us sources.ImportableDomain
	err := c.ShouldBindJSON(&us)
	if err != nil {
		log.Printf("%s sends invalid ImportableDomain JSON: %w", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %w", err)})
		return
	}

	if len(us.FQDN) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Domain to act on not filled"})
		return
	}

	success, err := sr.ActionOnListedDomain(us.FQDN, us.BtnAction)
	status := http.StatusBadRequest
	if success {
		status = http.StatusOK
	}
	c.JSON(status, err.Error())
}

func getAvailableResourceTypes(c *gin.Context) {
	source := c.MustGet("source").(*happydns.SourceCombined)

	lrt, ok := source.Source.(sources.LimitedResourceTypesSource)
	if !ok {
		// Return all types known to be supported by happyDNS
		c.JSON(http.StatusOK, sources.DefaultAvailableTypes)
	}

	c.JSON(http.StatusOK, lrt.ListAvailableTypes())
}
