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

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type CheckerController struct {
	checkerService happydns.CheckerUsecase
}

func NewCheckerController(checkerService happydns.CheckerUsecase) *CheckerController {
	return &CheckerController{
		checkerService,
	}
}

// CheckerHandler is a middleware that retrieves a check by name and sets it in the context.
func (uc *CheckerController) CheckerHandler(c *gin.Context) {
	cname := c.Param("cname")

	checker, err := uc.checkerService.GetChecker(cname)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "Check not found"})
		return
	}

	c.Set("checker", checker)

	c.Next()
}

// CheckerOptionHandler is a middleware that retrieves a specific option and sets it in the context.
func (uc *CheckerController) CheckerOptionHandler(c *gin.Context) {
	cname := c.Param("cname")
	optname := c.Param("optname")

	opts, err := uc.checkerService.GetCheckerOptions(cname, nil, nil, nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Set("option", (*opts)[optname])

	c.Next()
}

// ListCheckers retrieves all available checks.
//
//	@Summary		List checkers (admin)
//	@Schemes
//	@Description	Returns a list of all available checks with their version information.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]happydns.CheckerResponse	"Map of checker names to info"
//	@Failure		500	{object}	happydns.ErrorResponse					"Internal server error"
//	@Router			/checks [get]
func (uc *CheckerController) ListCheckers(c *gin.Context) {
	checkers, err := uc.checkerService.ListCheckers()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	res := map[string]happydns.CheckerResponse{}

	for name, checker := range *checkers {
		res[name] = happydns.CheckerResponse{
			ID:           checker.ID(),
			Name:         checker.Name(),
			Availability: checker.Availability(),
		}
	}

	happydns.ApiResponse(c, res, nil)
}

// GetCheckerStatus retrieves the status and available options for a check.
//
//	@Summary		Get check info
//	@Schemes
//	@Description	Retrieves the status information and available options for a specific checker.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cname	path		string	true	"Checker name"
//	@Success		200		{object}	happydns.CheckerResponse	"Checker status with version info and available options"
//	@Failure		404		{object}	happydns.ErrorResponse	"Checker not found"
//	@Router			/checks/{cname} [get]
func (uc *CheckerController) GetCheckerStatus(c *gin.Context) {
	checker := c.MustGet("checker").(happydns.Checker)

	res := happydns.CheckerResponse{
		ID:           checker.ID(),
		Name:         checker.Name(),
		Availability: checker.Availability(),
	}

	c.JSON(http.StatusOK, res)
}

// GetCheckerOptions retrieves all options for a check.
//
//	@Summary		Get check options (admin)
//	@Schemes
//	@Description	Retrieves all configuration options for a specific check.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cname	path		string	true	"Checker name"
//	@Success		200		{object}	happydns.CheckerOptions	"Checker options as key-value pairs"
//	@Failure		404		{object}	happydns.ErrorResponse	"Checker not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/checks/{cname}/options [get]
func (uc *CheckerController) GetCheckerOptions(c *gin.Context) {
	cname := c.Param("cname")

	opts, err := uc.checkerService.GetCheckerOptions(cname, nil, nil, nil)
	happydns.ApiResponse(c, opts, err)
}

// AddCheckerOptions adds or overwrites specific admin-level options for a check.
//
//	@Summary		Add checker options
//	@Schemes
//	@Description	Adds or overwrites specific configuration options for a checker without affecting other options.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cname	path		string								true	"Checker name"
//	@Param			body	body		happydns.SetCheckerOptionsRequest	true	"Options to add or overwrite"
//	@Success		200		{object}	bool								"Success status"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid request body"
//	@Failure		404		{object}	happydns.ErrorResponse				"Checker not found"
//	@Failure		500		{object}	happydns.ErrorResponse				"Internal server error"
//	@Router			/checks/{cname}/options [post]
func (uc *CheckerController) AddCheckerOptions(c *gin.Context) {
	cname := c.Param("cname")

	var req happydns.SetCheckerOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = uc.checkerService.OverwriteSomeCheckerOptions(cname, nil, nil, nil, req.Options)
	happydns.ApiResponse(c, true, err)
}

// ChangeCheckerOptions replaces all options for a checker.
//
//	@Summary		Replace checker options (admin)
//	@Schemes
//	@Description	Replaces all configuration options for a check with the provided options.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cname	path		string								true	"Checker name"
//	@Param			body	body		happydns.SetCheckerOptionsRequest	true	"New complete set of options"
//	@Success		200		{object}	bool								"Success status"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid request body"
//	@Failure		404		{object}	happydns.ErrorResponse				"Checker not found"
//	@Failure		500		{object}	happydns.ErrorResponse				"Internal server error"
//	@Router			/checks/{cname}/options [put]
func (uc *CheckerController) ChangeCheckerOptions(c *gin.Context) {
	cname := c.Param("cname")

	var req happydns.SetCheckerOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = uc.checkerService.SetCheckerOptions(cname, nil, nil, nil, req.Options)
	happydns.ApiResponse(c, true, err)
}

// GetCheckerOption retrieves a specific option value for a checker.
//
//	@Summary		Get checker option (admin)
//	@Schemes
//	@Description	Retrieves the value of a specific configuration option for a checker.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cname		path		string	true	"Checker name"
//	@Param			optname		path		string	true	"Option name"
//	@Success		200			{object}	object	"Option value (type varies)"
//	@Failure		404			{object}	happydns.ErrorResponse	"Checker not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/checks/{cname}/options/{optname} [get]
func (uc *CheckerController) GetCheckerOption(c *gin.Context) {
	opt := c.MustGet("option")

	happydns.ApiResponse(c, opt, nil)
}

// SetCheckerOption sets or updates a specific option value for a checker.
//
//	@Summary		Set checker option (admin)
//	@Schemes
//	@Description	Sets or updates the value of a specific configuration option for a checker.
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			cname		path		string	true	"Checker name"
//	@Param			optname		path		string	true	"Option name"
//	@Param			body		body		object	true	"Option value (type varies by option)"
//	@Success		200			{object}	bool	"Success status"
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid request body"
//	@Failure		404			{object}	happydns.ErrorResponse	"Checker not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/checks/{cname}/options/{optname} [put]
func (uc *CheckerController) SetCheckerOption(c *gin.Context) {
	cname := c.Param("cname")
	optname := c.Param("optname")

	var req any
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	po := happydns.CheckerOptions{}
	po[optname] = req

	err = uc.checkerService.OverwriteSomeCheckerOptions(cname, nil, nil, nil, po)
	happydns.ApiResponse(c, true, err)
}
