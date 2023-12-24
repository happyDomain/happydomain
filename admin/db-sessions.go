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
