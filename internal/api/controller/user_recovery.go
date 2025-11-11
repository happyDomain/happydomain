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

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type UserRecoveryController struct {
	auService happydns.AuthUserUsecase
}

func NewUserRecoveryController(auService happydns.AuthUserUsecase) *UserRecoveryController {
	return &UserRecoveryController{
		auService: auService,
	}
}

// UserRecoveryOperations performs account recovery.
//
//	@Summary	Account recovery.
//	@Schemes
//	@Description	This will send an email to the user either to recover its account or with a new email validation link.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.UserSpecialAction	true	"Description of the action to perform and email of the user"
//	@Success		200		{object}	happydns.ErrorResponse		"Perhaps something happen"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [patch]
func (rc *UserRecoveryController) UserRecoveryOperations(c *gin.Context) {
	var uu happydns.UserSpecialAction
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		log.Printf("%s sends invalid User JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	res := happydns.ErrorResponse{Message: "If this address exists in our database, you'll receive a new e-mail."}

	user, err := rc.auService.GetAuthUserByEmail(string(uu.Email))
	if err != nil {
		log.Printf("%s: unable to retrieve user %q: %s", c.ClientIP(), uu.Email, err.Error())
	} else if uu.Kind == "recovery" {
		if user.EmailVerification == nil {
			if err = rc.auService.SendValidationLink(user); err != nil {
				log.Printf("%s: unable to SendValidationLink in specialUserOperations: %s", c.ClientIP(), err.Error())
				c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to send email validation link. Please try again later."})
				return
			}

			log.Printf("%s: Sent validation link to: %s", c.ClientIP(), user.Email)
		} else {
			if err = rc.auService.SendRecoveryLink(user); err != nil {
				log.Printf("%s: unable to SendRecoveryLink in specialUserOperations: %s", c.ClientIP(), err.Error())
				c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to send accont recovery link. Please try again later."})
				return
			}

			log.Printf("%s: Sent recovery link to: %s", c.ClientIP(), user.Email)
		}
	} else if uu.Kind == "validation" {
		// Email have already been validated, do nothing
		if user.EmailVerification != nil {
			c.JSON(http.StatusOK, res)
			return
		}

		if err = rc.auService.SendValidationLink(user); err != nil {
			log.Printf("%s: unable to SendValidationLink 2 in specialUserOperations: %s", c.ClientIP(), err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to sent email validation link. Please try again later."})
			return
		}

		log.Printf("%s: Sent validation link to: %s", c.ClientIP(), user.Email)
	}

	c.JSON(http.StatusOK, res)
}

// validateUserAddress validates a user address after registration.
//
//	@Summary	Validate e-mail address.
//	@Schemes
//	@Description	This is the route called by the web interface in order to validate the e-mail address of the user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path		string						true	"User identifier"
//	@Param			body	body		happydns.AddressValidationForm	true	"Validation form"
//	@Success		204		{null}		null						"Email validated, you can now login"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{userId}/email [post]
func (rc *UserRecoveryController) ValidateUserAddress(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var uav happydns.AddressValidationForm
	err := c.ShouldBindJSON(&uav)
	if err != nil {
		log.Printf("%s sends invalid AddressValidation JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := rc.auService.ValidateEmail(user, uav); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// recoverUserAccount performs account recovery by reseting the password of the account.
//
//	@Summary	Reset password with link in email.
//	@Schemes
//	@Description	This performs	account	recovery	by reseting the	password of the	account.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path		string					true	"User identifier"
//	@Param			body	body		happydns.AccountRecoveryForm	true	"Recovery form"
//	@Success		204		{null}		null					"Recovery completed, you can now login with your new credentials"
//	@Failure		400		{object}	happydns.ErrorResponse			"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{userId}/recovery [post]
func (rc *UserRecoveryController) RecoverUserAccount(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var uar happydns.AccountRecoveryForm
	err := c.ShouldBindJSON(&uar)
	if err != nil {
		log.Printf("%s sends invalid AccountRecovey JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := rc.auService.ResetPassword(user, uar); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	log.Printf("%s: User recovered: %s", c.ClientIP(), user.Email)
	c.Status(http.StatusNoContent)
}
