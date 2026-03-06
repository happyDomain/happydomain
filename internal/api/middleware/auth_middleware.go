// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package middleware

import (
	"fmt"
	"log"
	"net/http"

	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	gorillasessions "github.com/gorilla/sessions"

	"git.happydns.org/happyDomain/model"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("LoggedUser"); !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Please login to access this resource.", Link: "/login"})
			return
		}

		c.Next()
	}
}

// gorillasessionExposer is satisfied by the concrete gin-contrib/sessions
// type, which wraps a *gorilla/sessions.Session and exposes it via Session().
// Using a duck-typed local interface avoids importing gin-contrib internals.
type gorillasessionExposer interface {
	Session() *gorillasessions.Session
}

func SessionLoginOK(c *gin.Context, user happydns.UserInfo) error {
	session := ginsessions.Default(c)

	// Phase 1: invalidate the pre-login session to prevent session fixation.
	// Preserve the original session options (Secure flag, Path, MaxAge) so
	// we can restore them on the new session.
	// Setting MaxAge=-1 causes the store to delete the server-side record and
	// send an expired cookie on Save().
	var origOptions *gorillasessions.Options
	if gs, ok := session.(gorillasessionExposer); ok {
		if gs.Session().Options != nil {
			opts := *gs.Session().Options // copy by value
			origOptions = &opts
		}
	}

	session.Clear()
	session.Options(ginsessions.Options{MaxAge: -1})
	session.Save()

	// Phase 2: create a genuinely new session with a fresh ID.
	// Reset the gorilla session's ID so the store generates a new one,
	// then restore the original cookie options.
	if gs, ok := session.(gorillasessionExposer); ok {
		gs.Session().ID = ""
		if origOptions != nil {
			origOptions.MaxAge = 86400 * 30 // restore positive MaxAge
			gs.Session().Options = origOptions
		}
	}

	session.Set("iduser", user.GetUserId())
	err := session.Save()
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("failed to save user session: %s", err),
			UserMessage: "Invalid username or password.",
		}
	}

	log.Printf("%s: now logged as %q\n", c.ClientIP(), user.GetEmail())
	return nil
}
