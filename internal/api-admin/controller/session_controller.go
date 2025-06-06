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

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

type SessionController struct {
	config *happydns.Options
	store  session.SessionStorage
}

func NewSessionController(cfg *happydns.Options, store session.SessionStorage) *SessionController {
	return &SessionController{
		config: cfg,
		store:  store,
	}
}

func (sc *SessionController) DeleteSessions(c *gin.Context) {
	happydns.ApiResponse(c, true, sc.store.ClearSessions())
}

func (sc *SessionController) SessionHandler(c *gin.Context) {
	session, err := sc.store.GetSession(c.Param("sessionid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Set("session", session)

	c.Next()
}

func (sc *SessionController) GetSession(c *gin.Context) {
	c.JSON(http.StatusOK, c.MustGet("session"))
}

func (sc *SessionController) DeleteSession(c *gin.Context) {
	happydns.ApiResponse(c, true, sc.store.DeleteSession(c.Param("sessionid")))
}
