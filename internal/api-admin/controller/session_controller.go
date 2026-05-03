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

	happydns "git.happydns.org/happyDomain/model"
)

type SessionController struct {
	config         *happydns.Options
	sessionService happydns.AdminSessionUsecase
}

func NewSessionController(cfg *happydns.Options, sessionService happydns.AdminSessionUsecase) *SessionController {
	return &SessionController{
		config:         cfg,
		sessionService: sessionService,
	}
}

// deleteSessions removes all sessions from the system.
//
//	@Summary		Delete all sessions
//	@Schemes
//	@Description	Remove all sessions from the system.
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	bool
//	@Failure		500	{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/sessions [delete]
func (sc *SessionController) DeleteSessions(c *gin.Context) {
	happydns.ApiResponse(c, true, sc.sessionService.ClearAllSessions())
}

// sessionHandler is a middleware that loads a session by ID and adds it to the context.
//
//	@Summary		Load session middleware
//	@Schemes
//	@Description	Middleware that retrieves a session by ID and adds it to the request context.
//	@Tags			sessions
//	@Param			sessionid	path	string	true	"Session identifier"
//	@Failure		404			{object}	happydns.ErrorResponse	"Session not found"
func (sc *SessionController) SessionHandler(c *gin.Context) {
	session, err := sc.sessionService.GetSessionByID(c.Param("sessionid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Set("session", session)

	c.Next()
}

// getSession retrieves a specific session by ID.
//
//	@Summary		Retrieve session
//	@Schemes
//	@Description	Get details of a specific session by its identifier.
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			sessionid	path		string	true	"Session identifier"
//	@Success		200			{object}	happydns.Session
//	@Failure		404			{object}	happydns.ErrorResponse	"Session not found"
//	@Router			/sessions/{sessionid} [get]
func (sc *SessionController) GetSession(c *gin.Context) {
	c.JSON(http.StatusOK, c.MustGet("session"))
}

// deleteSession deletes a specific session by ID.
//
//	@Summary		Delete session
//	@Schemes
//	@Description	Remove a specific session from the system by its identifier.
//	@Tags			sessions
//	@Accept			json
//	@Produce		json
//	@Param			sessionid	path		string	true	"Session identifier"
//	@Success		200			{object}	bool
//	@Failure		404			{object}	happydns.ErrorResponse	"Session not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/sessions/{sessionid} [delete]
func (sc *SessionController) DeleteSession(c *gin.Context) {
	happydns.ApiResponse(c, true, sc.sessionService.DeleteSessionByID(c.Param("sessionid")))
}
