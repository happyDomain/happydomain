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
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// CheckerController handles user-scoped check operations for the main API.
// All methods work with options scoped to the authenticated user.
type CheckerController struct {
	*BaseCheckerController
}

func NewCheckerController(checkerService happydns.CheckerUsecase) *CheckerController {
	return &CheckerController{
		BaseCheckerController: NewBaseCheckerController(checkerService),
	}
}

// CheckerHandler is a middleware that retrieves a check by name and sets it in the context.
func (uc *CheckerController) CheckerHandler(c *gin.Context) {
	cname := c.Param("cid")

	check, err := uc.checkerService.GetChecker(cname)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "Check not found"})
		return
	}

	c.Set("checker", check)

	c.Next()
}

// CheckerOptionHandler is a middleware that retrieves a specific check option for the authenticated user and sets it in the context.
func (uc *CheckerController) CheckerOptionHandler(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	cname := c.Param("cid")
	optname := c.Param("optname")

	opts, err := uc.checkerService.GetCheckerOptions(cname, &user.Id, nil, nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Set("option", (*opts)[optname])

	c.Next()
}

// GetCheckOptions retrieves all options for a check for the authenticated user.
//
//	@Summary		Get check options
//	@Schemes
//	@Description	Retrieves all configuration options for a specific check for the authenticated user.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cid	path		string	true	"Check name"
//	@Success		200		{object}	happydns.CheckerOptions		"Check options as key-value pairs"
//	@Failure		404		{object}	happydns.ErrorResponse	"Check not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/checks/{cid}/options [get]
func (uc *CheckerController) GetCheckerOptions(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	cname := c.Param("cid")

	uc.GetCheckerOptionsWithScope(c, cname, &user.Id, nil, nil)
}

// AddCheckerOptions adds or overwrites specific options for a check for the authenticated user.
//
//	@Summary		Add check options
//	@Schemes
//	@Description	Adds or overwrites specific configuration options for a check for the authenticated user without affecting other options.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cid	path		string									true	"Check name"
//	@Param			body	body		happydns.SetCheckerOptionsRequest	true	"Options to add or overwrite"
//	@Success		200		{object}	bool								"Success status"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid request body"
//	@Failure		404		{object}	happydns.ErrorResponse				"Check not found"
//	@Failure		500		{object}	happydns.ErrorResponse				"Internal server error"
//	@Router			/checks/{cid}/options [post]
func (uc *CheckerController) AddCheckerOptions(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	cname := c.Param("cid")

	uc.AddCheckerOptionsWithScope(c, cname, &user.Id, nil, nil)
}

// ChangeCheckerOptions replaces all options for a check for the authenticated user.
//
//	@Summary		Replace check options
//	@Schemes
//	@Description	Replaces all configuration options for a check for the authenticated user with the provided options.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cid	path		string									true	"Checker name"
//	@Param			body	body		happydns.SetCheckerOptionsRequest	true	"New complete set of options"
//	@Success		200		{object}	bool								"Success status"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid request body"
//	@Failure		404		{object}	happydns.ErrorResponse				"Checker not found"
//	@Failure		500		{object}	happydns.ErrorResponse				"Internal server error"
//	@Router			/checks/{cid}/options [put]
func (uc *CheckerController) ChangeCheckerOptions(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	cname := c.Param("cid")

	uc.ChangeCheckerOptionsWithScope(c, cname, &user.Id, nil, nil)
}

// GetCheckerOption retrieves a specific option value for a check for the authenticated user.
//
//	@Summary		Get check option
//	@Schemes
//	@Description	Retrieves the value of a specific configuration option for a check for the authenticated user.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cid		path		string	true	"Check name"
//	@Param			optname		path		string	true	"Option name"
//	@Success		200			{object}	object	"Option value (type varies)"
//	@Failure		404			{object}	happydns.ErrorResponse	"Check not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/checks/{cid}/options/{optname} [get]
func (uc *CheckerController) GetCheckerOption(c *gin.Context) {
	uc.GetCheckerOptionValue(c)
}

// SetCheckerOption sets or updates a specific option value for a check for the authenticated user.
//
//	@Summary		Set check option
//	@Schemes
//	@Description	Sets or updates the value of a specific configuration option for a check for the authenticated user.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cid		path		string	true	"Check name"
//	@Param			optname		path		string	true	"Option name"
//	@Param			body		body		object	true	"Option value (type varies by option)"
//	@Success		200			{object}	bool	"Success status"
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid request body"
//	@Failure		404			{object}	happydns.ErrorResponse	"Check not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/checks/{cid}/options/{optname} [put]
func (uc *CheckerController) SetCheckerOption(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	cname := c.Param("cid")
	optname := c.Param("optname")

	uc.SetCheckerOptionWithScope(c, cname, optname, &user.Id, nil, nil)
}
