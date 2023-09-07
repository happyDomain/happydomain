// Copyright or © or Copr. happyDNS (2020)
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

package admin

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

func declareSessionsRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.DELETE("/sessions", deleteSessions)

	apiSessionsRoutes := router.Group("/sessions/:sessionid")
	apiSessionsRoutes.Use(sessionHandler)

	apiSessionsRoutes.GET("", getSession)
	apiSessionsRoutes.DELETE("", deleteSession)
}

func deleteSessions(c *gin.Context) {
	ApiResponse(c, true, storage.MainStore.ClearSessions())
}

func sessionHandler(c *gin.Context) {
	sessionid, err := base64.StdEncoding.DecodeString(c.Param("sessionid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	session, err := storage.MainStore.GetSession(sessionid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	c.Set("session", session)

	c.Next()
}

func getSession(c *gin.Context) {
	session := c.MustGet("session").(*happydns.Session)

	c.JSON(http.StatusOK, session)
}

func deleteSession(c *gin.Context) {
	session := c.MustGet("session").(*happydns.Session)

	ApiResponse(c, true, storage.MainStore.DeleteSession(session))
}
