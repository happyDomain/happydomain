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

package route

import (
	"github.com/gin-gonic/gin"

	api "git.happydns.org/happyDomain/internal/api/route"
	"git.happydns.org/happyDomain/internal/storage"
	happydns "git.happydns.org/happyDomain/model"
)

// Dependencies holds all use cases required to register the admin API routes.
type Dependencies struct {
	AuthUser              happydns.AuthUserUsecase
	Checker               happydns.CheckerUsecase
	Domain                happydns.DomainUsecase
	Provider              happydns.ProviderUsecase
	RemoteZoneImporter    happydns.RemoteZoneImporterUsecase
	Service               happydns.ServiceUsecase
	User                  happydns.UserUsecase
	Zone                  happydns.ZoneUsecase
	ZoneCorrectionApplier happydns.ZoneCorrectionApplierUsecase
	ZoneImporter          happydns.ZoneImporterUsecase
	ZoneService           happydns.ZoneServiceUsecase
}

func DeclareRoutes(cfg *happydns.Options, router *gin.Engine, s storage.Storage, dep Dependencies) {
	apiRoutes := router.Group("/api")

	declareBackupRoutes(cfg, apiRoutes, s)
	declareDomainRoutes(apiRoutes, dep, s)
	declareChecksRoutes(apiRoutes, dep)
	declareProviderRoutes(apiRoutes, dep, s)
	declareSessionsRoutes(cfg, apiRoutes, s)
	declareUserAuthsRoutes(apiRoutes, dep, s)
	declareUsersRoutes(apiRoutes, dep, s)
	declareTidyRoutes(apiRoutes, s)
	api.DeclareVersionRoutes(apiRoutes)
}
