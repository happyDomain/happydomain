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
	"git.happydns.org/happyDomain/internal/usecase"
)

func DeclareKeystoreRoutes(router *gin.RouterGroup, keystores *usecase.KeystoreUsecase) {
	kc := controller.NewKeystoreController(keystores)

	router.GET("/keystores", kc.ListKeystores)
	// Limited, as creating some kinds checks the login password or derives
	// a key, slow by design.
	router.POST("/keystores", perClientRateLimiter(10), kc.CreateKeystore)
	router.GET("/keystores/kinds", kc.ListKeystoreKinds)
	router.PUT("/keystores/default", kc.SetDefaultKeystore)
	// Limited, as updating some kinds asks a remote service.
	router.PUT("/keystores/:ref", perClientRateLimiter(10), kc.UpdateKeystore)
	router.DELETE("/keystores/:ref", kc.DeleteKeystore)
	router.POST("/keystores/:ref/migrate", kc.MigrateKeystore)
	// Limited, so that what unlocks a keystore is not guessed.
	router.POST("/keystores/:ref/unlock", perClientRateLimiter(10), kc.UnlockKeystore)
	router.POST("/keystores/:ref/lock", kc.LockKeystore)
}
