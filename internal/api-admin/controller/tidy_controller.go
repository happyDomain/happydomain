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
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type TidyController struct {
	tidyUpService happydns.TidyUpUseCase
}

func NewTidyController(tidyUpService happydns.TidyUpUseCase) *TidyController {
	return &TidyController{
		tidyUpService: tidyUpService,
	}
}

// tidyDB performs database cleanup and maintenance operations.
//
//	@Summary	Tidy up the database
//	@Schemes
//	@Description	Performs cleanup and maintenance operations on the database, removing orphaned records and optimizing storage.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{boolean}	bool
//	@Failure	500	{object}	happydns.ErrorResponse	"Internal server error"
//	@Router		/tidy [post]
func (tc *TidyController) TidyDB(c *gin.Context) {
	happydns.ApiResponse(c, true, tc.tidyUpService.TidyAll())
}
