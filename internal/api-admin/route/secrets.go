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

	"git.happydns.org/happyDomain/internal/api-admin/controller"
)

func declareSecretsRoutes(router *gin.RouterGroup, dep Dependencies) {
	sc := controller.NewSecretController(dep.Secrets)

	router.GET("/secrets/status", sc.Status)
	router.POST("/secrets/rewrap", sc.Rewrap)
	router.POST("/secrets/reseal", sc.Reseal)
	router.POST("/secrets/drop-safes", sc.DropSafes)

	if dep.Keystores == nil {
		return
	}
	kc := controller.NewKeystoreController(dep.Keystores)
	router.GET("/secrets/keystore-kinds", kc.ListKeystoreKinds)
	owner := router.Group("/secrets/owners/:owner", kc.OwnerHandler)
	owner.GET("/keystores", kc.ListKeystores)
	owner.POST("/keystores", kc.CreateKeystore)
	owner.PUT("/keystores/default", kc.SetDefaultKeystore)
	owner.DELETE("/keystores/:ref", kc.DeleteKeystore)
	owner.POST("/keystores/:ref/migrate", kc.MigrateKeystore)
}
