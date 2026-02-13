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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type RegistrationController struct {
	auService happydns.AuthUserUsecase
	captcha   happydns.CaptchaVerifier
}

func NewRegistrationController(auService happydns.AuthUserUsecase, captchaVerifier happydns.CaptchaVerifier) *RegistrationController {
	return &RegistrationController{
		auService: auService,
		captcha:   captchaVerifier,
	}
}

// RegisterNewUser checks and appends a user in the database.
//
//	@Summary	Register account.
//	@Schemes
//	@Description	Register a new happyDomain account (when using internal authentication system).
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.UserRegistration	true	"Account information"
//	@Success		200		{object}	happydns.User		"The created user"
//	@Failure		400		{object}	happydns.ErrorResponse		"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [post]
func (rc *RegistrationController) RegisterNewUser(c *gin.Context) {
	var uu happydns.UserRegistration
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		log.Printf("%s sends invalid User JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if rc.captcha.Provider() != "" {
		if err := rc.captcha.Verify(uu.CaptchaToken, c.ClientIP()); err != nil {
			log.Printf("%s: captcha verification failed during registration: %s", c.ClientIP(), err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Captcha verification failed."})
			return
		}
	}

	err = rc.auService.CanRegister(uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	user, err := rc.auService.CreateAuthUser(uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	log.Printf("%s: registers new user: %s", c.ClientIP(), user.Email)

	c.JSON(http.StatusOK, user)
}
