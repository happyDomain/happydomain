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

	"git.happydns.org/happyDomain/api/controller"
	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/model"
)

func DeclareUsersRoutes(router *gin.RouterGroup, dependancies happydns.UsecaseDependancies, lc *controller.LoginController) *controller.UserController {
	uc := controller.NewUserController(dependancies.GetUserUsecase(), lc)

	apiUserRoutes := router.Group("/users/:uid")
	apiUserRoutes.Use(middleware.UserHandler(dependancies.GetUserUsecase()))

	apiUserRoutes.GET("", uc.GetUser)
	apiUserRoutes.GET("/avatar.png", uc.GetUserAvatar)

	apiSameUserRoutes := router.Group("/users/:uid")
	apiSameUserRoutes.Use(middleware.UserHandler(dependancies.GetUserUsecase()))
	apiSameUserRoutes.Use(middleware.SameUserHandler)

	apiSameUserRoutes.DELETE("", uc.DeleteMyUser)
	apiSameUserRoutes.GET("/settings", uc.GetUserSettings)
	apiSameUserRoutes.POST("/settings", uc.ChangeUserSettings)

	return uc
}
