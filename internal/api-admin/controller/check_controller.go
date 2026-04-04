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

	apicontroller "git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
)

// AdminCheckerController handles admin checker-related API endpoints.
// It embeds CheckerController and overrides GetCheckerOptions to return a flat
// (non-positional) map scoped to nil (global/admin) level.
type AdminCheckerController struct {
	*apicontroller.CheckerController
}

// NewAdminCheckerController creates a new AdminCheckerController.
func NewAdminCheckerController(optionsUC *checkerUC.CheckerOptionsUsecase) *AdminCheckerController {
	return &AdminCheckerController{
		CheckerController: apicontroller.NewCheckerController(nil, optionsUC, nil, nil, nil),
	}
}

// GetCheckerOptions returns admin-level options (nil scope) for a checker as a flat map.
//
//	@Summary	Get admin-level checker options
//	@Tags		admin,checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Success	200	{object}	checker.CheckerOptions
//	@Router		/checkers/{checkerId}/options [get]
func (cc *AdminCheckerController) GetCheckerOptions(c *gin.Context) {
	checkerID := c.Param("checkerId")
	opts, err := cc.OptionsUC.GetCheckerOptions(checkerID, nil, nil, nil)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, opts)
}
