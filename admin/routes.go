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

package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
)

func DeclareRoutes(cfg *config.Options, router *gin.Engine) {
	apiRoutes := router.Group("/api")

	declareBackupRoutes(cfg, apiRoutes)
	declareUserAuthsRoutes(cfg, apiRoutes)
	declareDomainsRoutes(cfg, apiRoutes)
	declareProvidersRoutes(cfg, apiRoutes)
	declareSessionsRoutes(cfg, apiRoutes)
	declareUsersRoutes(cfg, apiRoutes)
	api.DeclareVersionRoutes(apiRoutes)
}

func ApiResponse(c *gin.Context, data interface{}, err error) {
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
