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
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/helpers"
	happydns "git.happydns.org/happyDomain/model"
)

// UserStats holds aggregate resource counts for a single user.
type UserStats struct {
	User          *happydns.User `json:"user"`
	ProviderCount int            `json:"provider_count"`
	DomainCount   int            `json:"domain_count"`
	ZoneCount     int            `json:"zone_count"`
}

type UserController struct {
	userService      happydns.UserUsecase
	adminService     happydns.AdminUserUsecase
	authService      happydns.AuthUserUsecase
	authAdminService happydns.AdminAuthUserUsecase
	adminDomain      happydns.AdminDomainUsecase
	adminProvider    happydns.AdminProviderUsecase
}

func NewUserController(userService happydns.UserUsecase, adminService happydns.AdminUserUsecase, authService happydns.AuthUserUsecase, authAdminService happydns.AdminAuthUserUsecase, adminDomain happydns.AdminDomainUsecase, adminProvider happydns.AdminProviderUsecase) *UserController {
	return &UserController{
		userService:      userService,
		adminService:     adminService,
		authService:      authService,
		authAdminService: authAdminService,
		adminDomain:      adminDomain,
		adminProvider:    adminProvider,
	}
}

func (uc *UserController) UserHandler(c *gin.Context) {
	user, err := middleware.UserHandlerBase(uc.userService, c)
	if err != nil {
		user, err = uc.userService.GetUserByEmail(c.Param("uid"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "User not found"})
			return
		}
	}

	c.Set("user", user)

	c.Next()
}

// getUsers retrieves all users from the database.
//
//	@Summary		List all users.
//	@Schemes
//	@Description	Retrieve a list of all users in the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		happydns.User			"List of users"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [get]
func (uc *UserController) GetUsers(c *gin.Context) {
	users, err := uc.adminService.ListAllUsers()
	happydns.ApiResponse(c, users, err)
}

// newUser creates a new user in the database.
//
//	@Summary		Create a new user.
//	@Schemes
//	@Description	Create a new user account with the provided information.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.User			true	"User information"
//	@Success		200		{object}	happydns.User			"The created user"
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [post]
func (uc *UserController) NewUser(c *gin.Context) {
	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	happydns.ApiResponse(c, uu, uc.adminService.CreateOrUpdateUser(uu))
}

// deleteUsers deletes all users from the database.
//
//	@Summary		Delete all users.
//	@Schemes
//	@Description	Remove all user accounts from the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200		{boolean}	bool					"Success status"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [delete]
func (uc *UserController) DeleteUsers(c *gin.Context) {
	happydns.ApiResponse(c, true, uc.adminService.ClearUsers())
}

// getUser retrieves a specific user from the database.
//
//	@Summary		Show user.
//	@Schemes
//	@Description	Retrieve a user's complete information by their ID or email.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Success		200		{object}	happydns.User			"The user"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Router			/users/{uid} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user)
}

// updateUser updates an existing user's information.
//
//	@Summary		Update user.
//	@Schemes
//	@Description	Update a user's information. The user ID is preserved from the URL parameter.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Param			body	body		happydns.User			true	"Updated user information"
//	@Success		200		{object}	happydns.User			"The updated user"
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{uid} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = user.Id

	updated, err := uc.userService.UpdateUser(uu.Id, func(u *happydns.User) {
		// Stamp quota update time if quota fields changed.
		if uu.Quota != u.Quota {
			uu.Quota.UpdatedAt = time.Now()
		}

		u.Email = uu.Email
		u.CreatedAt = uu.CreatedAt
		u.LastSeen = uu.LastSeen
		u.Settings = uu.Settings
		u.Quota = uu.Quota
	})

	happydns.ApiResponse(c, updated, err)
}

// deleteUser removes a specific user from the database.
//
//	@Summary		Delete user.
//	@Schemes
//	@Description	Delete a user account and all associated data.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Success		200		{boolean}	bool					"Success status"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{uid} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	happydns.ApiResponse(c, true, uc.adminService.DeleteUserByID(user.Id))
}

type newAuthUserResponse struct {
	Password string             `json:"password"`
	AuthUser *happydns.UserAuth `json:"authUser"`
}

// NewAuthUser creates a UserAuth for the given existing User.
//
//	@Summary		Create auth account for a user.
//	@Schemes
//	@Description	Generate a UserAuth for the given existing User, using the user's email and a freshly generated password returned in the response.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Success		200		{object}	newAuthUserResponse		"Generated password and created auth user"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{uid}/new_auth [post]
func (uc *UserController) NewAuthUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	password, err := helpers.GeneratePassword()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	au, err := happydns.NewUserAuth(user.Email, password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}
	au.Id = user.Id

	if existing, err := uc.authService.GetAuthUser(user.Id); err == nil && existing != nil {
		c.AbortWithStatusJSON(http.StatusConflict, happydns.ErrorResponse{Message: "An auth account already exists for this user."})
		return
	}
	if existing, err := uc.authService.GetAuthUserByEmail(user.Email); err == nil && existing != nil {
		c.AbortWithStatusJSON(http.StatusConflict, happydns.ErrorResponse{Message: "An auth account with this email already exists."})
		return
	}

	// AdminUpdateAuthUser persists at the supplied Id (Put), unlike
	// AdminCreateAuthUser which always allocates a fresh identifier.
	if err := uc.authAdminService.AdminUpdateAuthUser(au); err != nil {
		happydns.ApiResponse(c, nil, err)
		return
	}

	happydns.ApiResponse(c, newAuthUserResponse{Password: password, AuthUser: au}, nil)
}

// getUsersStats returns resource counts (providers, domains, zones) for each user.
//
//	@Summary		User resource stats.
//	@Schemes
//	@Description	Retrieve provider, domain and zone counts for every user.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		controller.UserStats	"Per-user statistics"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/stats [get]
func (uc *UserController) GetUserStats(c *gin.Context) {
	users, err := uc.adminService.ListAllUsers()
	if err != nil {
		happydns.ApiResponse(c, nil, err)
		return
	}

	domainCount := map[string]int{}
	zoneCount := map[string]int{}
	domains, err := uc.adminDomain.ListAllDomains()
	if err != nil {
		happydns.ApiResponse(c, nil, err)
		return
	}
	for _, d := range domains {
		key := d.Owner.String()
		domainCount[key]++
		zoneCount[key] += len(d.ZoneHistory)
	}

	providerCount := map[string]int{}
	providers, err := uc.adminProvider.ListAllProviderMetas()
	if err != nil {
		happydns.ApiResponse(c, nil, err)
		return
	}
	for _, p := range providers {
		providerCount[p.Owner.String()]++
	}

	stats := make([]UserStats, 0, len(users))
	for _, u := range users {
		key := u.Id.String()
		stats = append(stats, UserStats{
			User:          u,
			ProviderCount: providerCount[key],
			DomainCount:   domainCount[key],
			ZoneCount:     zoneCount[key],
		})
	}

	happydns.ApiResponse(c, stats, nil)
}
