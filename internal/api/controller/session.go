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
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type SessionController struct {
	sessionService happydns.SessionUsecase
}

func NewSessionController(sessionService happydns.SessionUsecase) *SessionController {
	return &SessionController{
		sessionService: sessionService,
	}
}

func (sc *SessionController) SessionHandler(c *gin.Context) {
	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	session, err := sc.sessionService.GetUserSession(myuser, c.Param("sid"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.Set("session", session)

	c.Next()
}

// getSession gets the content of the current user's session.
//
//	@Summary	Retrieve user's session content
//	@Schemes
//	@Description	Get the content of the current user's session.
//	@Tags			users
//	@Accept			json
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/session [get]
func (sc *SessionController) GetSession(c *gin.Context) {
	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}
	session := sessions.Default(c)

	s, err := sc.sessionService.GetUserSession(myuser, session.ID())
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, s)
}

// clearSession removes the content of the current user's session.
//
//	@Summary	Remove user's session content
//	@Schemes
//	@Description	Remove the content of the current user's session.
//	@Tags			users
//	@Accept			json
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		204	{null}		null
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/session [delete]
func (sc *SessionController) ClearSession(c *gin.Context) {
	session := sessions.Default(c)

	session.Clear()

	c.Status(http.StatusNoContent)
}

// clearUserSessions closes all existing sessions for a given user, and disconnect it.
//
//	@Summary	Remove user's sessions
//	@Schemes
//	@Description	Closes all sessions for a given user.
//	@Tags			users
//	@Accept			json
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		204	{null}		null
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/sessions [delete]
func (sc *SessionController) ClearUserSessions(c *gin.Context) {
	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	err := sc.sessionService.CloseUserSessions(myuser)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// getSessions lists the sessions open for the current user.
//
//	@Summary	List user's sessions
//	@Schemes
//	@Description	List the sessions open for the current user
//	@Tags			users
//	@Accept			json
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/sessions [get]
func (sc *SessionController) GetSessions(c *gin.Context) {
	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	s, err := sc.sessionService.ListUserSessions(myuser)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, s)
}

// createSession create a new session for the current user
//
//	@Summary	Create a new session for the current user.
//	@Schemes
//	@Description	Create a new session for the current user.
//	@Tags			users
//	@Accept			json
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/sessions [post]
func (sc *SessionController) CreateSession(c *gin.Context) {
	var us happydns.Session
	err := c.ShouldBindJSON(&us)
	if err != nil {
		log.Printf("%s sends invalid Session JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	sess, err := sc.sessionService.CreateUserSession(myuser, us.Description)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": sess.Id})
}

// updateSession update a session owned by the current user
//
//	@Summary	Update a session owned by the current user.
//	@Schemes
//	@Description	Update	a session owned	by the current user.
//	@Tags			users
//	@Accept			json
//	@Param			sessionId	path	string	true	"Session identifier"
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/sessions/{sessionId} [put]
func (sc *SessionController) UpdateSession(c *gin.Context) {
	var us happydns.Session
	err := c.ShouldBindJSON(&us)
	if err != nil {
		log.Printf("%s sends invalid Session JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	s, err := sc.sessionService.GetUserSession(myuser, c.Param("sid"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	err = sc.sessionService.UpdateUserSession(myuser, c.Param("sid"), func(newsession *happydns.Session) {
		newsession.Description = us.Description
		newsession.ExpiresOn = us.ExpiresOn
	})
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, s)
}

// deleteSession delete a session owned by the current user
//
//	@Summary	Delete a session owned by the current user.
//	@Schemes
//	@Description	Delete	a session owned	by the current user.
//	@Tags			users
//	@Accept			json
//	@Param			sessionId	path		string					true	"Session identifier"
//	@Prodsce		json
//	@Security		securitydefinitions.basic
//	@Ssccess		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/sessions/{sessionId} [delete]
func (sc *SessionController) DeleteSession(c *gin.Context) {
	myuser := middleware.MyUser(c)
	if myuser == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	err := sc.sessionService.DeleteUserSession(myuser, c.Param("sid"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, nil)
}
