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

package route

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/model"
)

// DeclareDomainWatchRoutes declares the routes of the watchlist. Adding a
// domain and checking it now query the registries while the user waits, so
// they go through lookupRL.
func DeclareDomainWatchRoutes(apiAuthRoutes *gin.RouterGroup, lookupRL gin.HandlerFunc, domainWatchUC happydns.DomainWatchUsecase) {
	wc := controller.NewDomainWatchController(domainWatchUC)

	watchlist := apiAuthRoutes.Group("/watchlist")
	watchlist.GET("", wc.ListDomainWatches)
	watchlist.POST("", lookupRL, wc.AddDomainWatch)

	watch := watchlist.Group("/:watchId")
	watch.GET("", wc.GetDomainWatch)
	watch.PUT("", wc.UpdateDomainWatch)
	watch.DELETE("", wc.DeleteDomainWatch)
	watch.POST("/check", lookupRL, wc.CheckDomainWatch)
}
