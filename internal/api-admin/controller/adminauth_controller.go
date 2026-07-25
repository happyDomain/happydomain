// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/adminauth"
	happydns "git.happydns.org/happyDomain/model"
)

type AdminAuthController struct {
	cfg      *happydns.Options
	throttle *adminauth.LoginThrottle
}

func NewAdminAuthController(cfg *happydns.Options) *AdminAuthController {
	return &AdminAuthController{cfg: cfg, throttle: adminauth.NewLoginThrottle()}
}

type adminLoginRequest struct {
	// Password is the admin password checked against the configured
	// AdminPasswordHash.
	Password string `json:"password"`

	// Duration is the requested session lifetime in seconds. It is clamped
	// server-side; 0 falls back to the default lifetime.
	Duration int `json:"duration"`
}

type adminLoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Login verifies the admin password and issues a short-lived admin session
// token.
//
//	@Summary		Admin login
//	@Schemes
//	@Description	Verify the admin password and return a short-lived bearer token for the admin interface.
//	@Tags			admin-auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		adminLoginRequest	true	"Admin credentials"
//	@Success		200		{object}	adminLoginResponse
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		401		{object}	happydns.ErrorResponse
//	@Failure		429		{object}	happydns.ErrorResponse
//	@Failure		503		{object}	happydns.ErrorResponse
//	@Router			/admin-login [post]
func (ac *AdminAuthController) Login(c *gin.Context) {
	// When no password is configured, admin authentication is disabled and
	// there is nothing to log into.
	if ac.cfg.AdminPasswordHash == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "Admin authentication is not enabled."})
		return
	}

	// This endpoint is the only unauthenticated one of the admin API, and the
	// only expensive one: verification is a deliberately slow password hash.
	// Both properties have to be defended before any work is done, otherwise the
	// endpoint doubles as a password-guessing oracle and as a way for an
	// anonymous caller to burn every core of a process that also serves the
	// public API.
	client := c.ClientIP()

	if retryAfter, ok := ac.throttle.Allow(client); !ok {
		c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, happydns.ErrorResponse{Message: "Too many login attempts, please retry later."})
		return
	}

	// Cap the body before decoding it: nothing legitimate needs more, and the
	// admin engine installs no global body limit.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, adminauth.MaxLoginBodyBytes)

	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ac.throttle.RecordFailure(client)
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid request."})
		return
	}

	release, ok := ac.throttle.Acquire()
	if !ok {
		c.Header("Retry-After", "5")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, happydns.ErrorResponse{Message: "Too many login attempts in flight, please retry later."})
		return
	}

	valid := adminauth.VerifyAdminPassword(ac.cfg.AdminPasswordHash, req.Password)
	release()

	if !valid {
		if lockedFor := ac.throttle.RecordFailure(client); lockedFor > 0 {
			log.Printf("%s: failed admin login attempt, client locked out for %s", client, lockedFor)
		} else {
			log.Printf("%s: failed admin login attempt", client)
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Invalid password."})
		return
	}

	ac.throttle.RecordSuccess(client)

	token, expiresAt, err := adminauth.IssueAdminToken(ac.cfg.JWTSecretKey, ac.cfg.AdminPasswordHash, time.Duration(req.Duration)*time.Second)
	if err != nil {
		log.Printf("%s: unable to issue admin token: %s", client, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Unable to issue admin session."})
		return
	}

	c.JSON(http.StatusOK, adminLoginResponse{Token: token, ExpiresAt: expiresAt})
}
