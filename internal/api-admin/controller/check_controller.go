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

	var res map[string]happydns.CheckerResponse

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
