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

	"git.happydns.org/happyDomain/api-admin/controller"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

func declareUserAuthsRoutes(router *gin.RouterGroup, dependancies happydns.UsecaseDependancies, store storage.Storage) {
	ac := controller.NewAuthUserController(dependancies.AuthUserUsecase(), store)

	router.GET("/auth", ac.GetAuthUsers)
	router.POST("/auth", ac.NewAuthUser)
	router.DELETE("/auth", ac.DeleteAuthUsers)

	apiUsersRoutes := router.Group("/auth/:uid")
	apiUsersRoutes.Use(ac.AuthUserHandler)

	apiUsersRoutes.GET("", ac.GetAuthUser)
	apiUsersRoutes.PUT("", ac.UpdateAuthUser)
	apiUsersRoutes.DELETE("", ac.DeleteAuthUser)

	apiUsersRoutes.POST("/recover_link", ac.RecoverUserAcct)
	apiUsersRoutes.POST("/reset_password", ac.ResetUserPasswd)
	apiUsersRoutes.POST("/send_recover_email", ac.SendRecoverUserAcct)
	apiUsersRoutes.POST("/send_validation_email", ac.SendValidateUserEmail)
	apiUsersRoutes.POST("/validation_link", ac.EmailValidationLink)
	apiUsersRoutes.POST("/validate_email", ac.ValidateEmail)
}
