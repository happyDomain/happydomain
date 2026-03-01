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

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// BaseCheckerController contains shared functionality for check controllers.
// It provides common methods that can be used by both admin and user-scoped controllers.
type BaseCheckerController struct {
	checkerService happydns.CheckerUsecase
}

func NewBaseCheckerController(checkerService happydns.CheckerUsecase) *BaseCheckerController {
	return &BaseCheckerController{
		checkerService,
	}
}

// GetCheckerService returns the check service for use by derived controllers.
func (bc *BaseCheckerController) GetCheckerService() happydns.CheckerUsecase {
	return bc.checkerService
}

// ListCheckers retrieves all available checks.
func (bc *BaseCheckerController) ListCheckers(c *gin.Context) {
	checkers, err := bc.checkerService.ListCheckers()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	res := map[string]happydns.CheckerResponse{}

	for name, checker := range *checkers {
		_, hasHTML := checker.(happydns.CheckerHTMLReporter)
		res[name] = happydns.CheckerResponse{
			ID:            name,
			Name:          checker.Name(),
			Availability:  checker.Availability(),
			Options:       checker.Options(),
			HasHTMLReport: hasHTML,
		}
	}

	happydns.ApiResponse(c, res, nil)
}

// GetCheckerStatus retrieves the status and available options for a check.
func (bc *BaseCheckerController) GetCheckerStatus(c *gin.Context) {
	checker := c.MustGet("checker").(happydns.Checker)

	_, hasHTML := checker.(happydns.CheckerHTMLReporter)
	c.JSON(http.StatusOK, happydns.CheckerResponse{
		ID:            checker.ID(),
		Name:          checker.Name(),
		Availability:  checker.Availability(),
		Options:       checker.Options(),
		HasHTMLReport: hasHTML,
	})
}

// GetCheckerOptionsWithScope retrieves all options for a check with the given scope.
func (bc *BaseCheckerController) GetCheckerOptionsWithScope(c *gin.Context, cname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	opts, err := bc.checkerService.GetCheckerOptions(cname, userId, domainId, serviceId)
	happydns.ApiResponse(c, opts, err)
}

// AddCheckerOptionsWithScope adds or overwrites specific options for a check with the given scope.
func (bc *BaseCheckerController) AddCheckerOptionsWithScope(c *gin.Context, cname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	var req happydns.SetCheckerOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = bc.checkerService.OverwriteSomeCheckerOptions(cname, userId, domainId, serviceId, req.Options)
	happydns.ApiResponse(c, true, err)
}

// ChangeCheckerOptionsWithScope replaces all options for a check with the given scope.
func (bc *BaseCheckerController) ChangeCheckerOptionsWithScope(c *gin.Context, cname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	var req happydns.SetCheckerOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = bc.checkerService.SetCheckerOptions(cname, userId, domainId, serviceId, req.Options)
	happydns.ApiResponse(c, true, err)
}

// GetCheckerOptionValue retrieves a specific option value from the context.
func (bc *BaseCheckerController) GetCheckerOptionValue(c *gin.Context) {
	opt := c.MustGet("option")

	happydns.ApiResponse(c, opt, nil)
}

// SetCheckerOptionWithScope sets or updates a specific option value for a check with the given scope.
func (bc *BaseCheckerController) SetCheckerOptionWithScope(c *gin.Context, cname string, optname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	var req any
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	po := happydns.CheckerOptions{}
	po[optname] = req

	err = bc.checkerService.OverwriteSomeCheckerOptions(cname, userId, domainId, serviceId, po)
	happydns.ApiResponse(c, true, err)
}
