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
	happydns "git.happydns.org/happyDomain/model"
)

// DeclareEmailAutoconfigRoutes wires the public HTTP endpoints for mail-client
// auto-configuration: the well-known XML paths dictated by the standards
// (Mozilla and Microsoft).
func DeclareEmailAutoconfigRoutes(baseRoutes *gin.RouterGroup, rl gin.HandlerFunc, uc happydns.EmailAutoconfigUsecase) {
	if uc == nil {
		return
	}

	ctrl := controller.NewEmailAutoconfigController(uc)

	// Mozilla Autoconfig: clients fetch GET https://autoconfig.<domain>/mail/config-v1.1.xml
	baseRoutes.GET("/mail/config-v1.1.xml", rl, ctrl.MozillaAutoconfig)

	// Microsoft Autodiscover: Outlook hits both GET and POST, with two
	// common spellings of the path.
	for _, path := range []string{
		"/Autodiscover/Autodiscover.xml",
		"/autodiscover/autodiscover.xml",
		"/AutoDiscover/AutoDiscover.xml",
	} {
		baseRoutes.GET(path, rl, ctrl.MSAutodiscover)
		baseRoutes.POST(path, rl, ctrl.MSAutodiscover)
	}
}
