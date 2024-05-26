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

package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/rrivera/identicon"

	"git.happydns.org/happyDomain/actions"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

func declareUsersRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.POST("/users", func(c *gin.Context) {
		registerUser(opts, c)
	})
	router.PATCH("/users", func(c *gin.Context) {
		specialUserOperations(opts, c)
	})

	apiUserRoutes := router.Group("/users/:uid")
	apiUserRoutes.Use(userAuthHandler)

	apiUserRoutes.POST("/email", validateUserAddress)
	apiUserRoutes.POST("/recovery", recoverUserAccount)
}

func declareUsersAuthRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.GET("/session", getSession)
	router.DELETE("/session", clearSession)

	router.GET("/sessions", getSessions)
	router.POST("/sessions", createSession)
	router.DELETE("/sessions", clearUserSessions)
	apiSessionsRoutes := router.Group("/sessions/:sid")
	apiSessionsRoutes.Use(sessionHandler)
	apiSessionsRoutes.PUT("", updateSession)
	apiSessionsRoutes.DELETE("", deleteSession)

	apiUserRoutes := router.Group("/users/:uid")
	apiUserRoutes.Use(userHandler)

	apiUserRoutes.GET("", getUser)
	apiUserRoutes.GET("/avatar.png", getUserAvatar)

	apiSameUserRoutes := router.Group("/users/:uid")
	apiSameUserRoutes.Use(userHandler)
	apiSameUserRoutes.Use(SameUserHandler)

	apiSameUserRoutes.DELETE("", func(c *gin.Context) {
		deleteMyUser(opts, c)
	})
	apiSameUserRoutes.GET("/settings", getUserSettings)
	apiSameUserRoutes.POST("/settings", changeUserSettings)

	apiUserAuthRoutes := router.Group("/users/:uid")
	apiUserAuthRoutes.Use(userAuthHandler)
	apiUserAuthRoutes.GET("/is_auth_user", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	apiUserAuthRoutes.POST("/delete", func(c *gin.Context) {
		deleteUser(opts, c)
	})
	apiUserAuthRoutes.POST("/new_password", func(c *gin.Context) {
		changePassword(opts, c)
	})
}

func myUser(c *gin.Context) (user *happydns.User) {
	if u, exists := c.Get("LoggedUser"); exists {
		user = u.(*happydns.User)
	} else if u, exists := c.Get("user"); exists {
		user = u.(*happydns.User)
	}
	return
}

type UserRegistration struct {
	Email      string
	Password   string
	Language   string `json:"lang,omitempty"`
	Newsletter bool   `json:"wantReceiveUpdate,omitempty"`
}

// registerUser checks and appends a user in the database.
//
//	@Summary	Register account.
//	@Schemes
//	@Description	Register a new happyDomain account (when using internal authentication system).
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UserRegistration	true	"Account information"
//	@Success		200		{object}	happydns.User		"The created user"
//	@Failure		400		{object}	happydns.Error		"Invalid input"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users [post]
func registerUser(opts *config.Options, c *gin.Context) {
	if opts.DisableRegistration {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Registration are closed on this instance."})
		return
	}

	var uu UserRegistration
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		log.Printf("%s sends invalid User JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if len(uu.Email) <= 3 || strings.Index(uu.Email, "@") == -1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "The given email is invalid."})
		return
	}

	if len(uu.Password) <= 7 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "The given password is invalid."})
		return
	}

	if storage.MainStore.AuthUserExists(uu.Email) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "An account already exists with the given address. Try login now."})
		return
	}

	user, err := happydns.NewUserAuth(uu.Email, uu.Password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	user.AllowCommercials = uu.Newsletter

	if err := storage.MainStore.CreateAuthUser(user); err != nil {
		log.Printf("%s: unable to CreateUser in registerUser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create your account. Please try again later."})
		return
	}

	if actions.SendValidationLink(opts, user); err != nil {
		log.Printf("%s: unable to SendValidationLink in registerUser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to send email validation link. Please try again later."})
		return
	}

	log.Printf("%s: registers new user: %s", c.ClientIP(), user.Email)

	c.JSON(http.StatusOK, user)
}

type UserSpecialAction struct {
	// Kind of special action to perform: "recovery" or "validation".
	Kind string

	// Email on which to perform actions.
	Email string
}

// specialUserOperations performs account recovery.
//
//	@Summary	Account recovery.
//	@Schemes
//	@Description	This will send an email to the user either to recover its account or with a new email validation link.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UserSpecialAction	true	"Description of the action to perform and email of the user"
//	@Success		200		{object}	happydns.Error		"Perhaps something happen"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users [patch]
func specialUserOperations(opts *config.Options, c *gin.Context) {
	var uu UserSpecialAction
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		log.Printf("%s sends invalid User JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	res := gin.H{"errmsg": "If this address exists in our database, you'll receive a new e-mail."}

	if user, err := storage.MainStore.GetAuthUserByEmail(uu.Email); err != nil {
		log.Printf("%s: unable to retrieve user %q: %s", c.ClientIP(), uu.Email, err.Error())
		c.JSON(http.StatusOK, res)
		return
	} else {
		if uu.Kind == "recovery" {
			if user.EmailVerification == nil {
				if err = actions.SendValidationLink(opts, user); err != nil {
					log.Printf("%s: unable to SendValidationLink in specialUserOperations: %s", c.ClientIP(), err.Error())
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to send email validation link. Please try again later."})
					return
				}

				log.Printf("%s: Sent validation link to: %s", c.ClientIP(), user.Email)
			} else {
				if err = actions.SendRecoveryLink(opts, user); err != nil {
					log.Printf("%s: unable to SendRecoveryLink in specialUserOperations: %s", c.ClientIP(), err.Error())
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to send accont recovery link. Please try again later."})
					return
				}

				if err := storage.MainStore.UpdateAuthUser(user); err != nil {
					log.Printf("%s: unable to UpdateUser in specialUserOperations: %s", c.ClientIP(), err.Error())
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
					return
				}

				log.Printf("%s: Sent recovery link to: %s", c.ClientIP(), user.Email)
			}
		} else if uu.Kind == "validation" {
			// Email have already been validated, do nothing
			if user.EmailVerification != nil {
				c.JSON(http.StatusOK, res)
				return
			}

			if err = actions.SendValidationLink(opts, user); err != nil {
				log.Printf("%s: unable to SendValidationLink 2 in specialUserOperations: %s", c.ClientIP(), err.Error())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to sent email validation link. Please try again later."})
				return
			}

			log.Printf("%s: Sent validation link to: %s", c.ClientIP(), user.Email)
		}
	}

	c.JSON(http.StatusOK, res)
}

func SameUserHandler(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)
	user := c.MustGet("user").(*happydns.User)

	if !bytes.Equal(user.Id, myuser.Id) {
		log.Printf("%s: tries to do action as %s (logged %s)", c.ClientIP(), myuser.Id, user.Id)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Not authorized"})
		return
	}

	c.Next()
}

// getUser shows a user in the database.
//
//	@Summary	Show user.
//	@Schemes
//	@Description	Show a user from the database, information is limited to id and email if this is not the current user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	happydns.User		"The created user"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users/{userId} [get]
func getUser(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)
	user := c.MustGet("user").(*happydns.User)

	if bytes.Equal(user.Id, myuser.Id) {
		c.JSON(http.StatusOK, user)
	} else {
		c.JSON(http.StatusOK, &happydns.User{
			Id:    user.Id,
			Email: user.Email,
		})
	}
}

// getUserAvatar returns a unique avatar for the user.
//
//	@Summary	Show user's avatar.
//	@Schemes
//	@Description	Returns a unique avatar for the user.
//	@Tags			users
//	@Accept			json
//	@Produce		png
//	@Param			size	query	int	false	"Image output desired size"
//	@Success		200		{file}		png
//	@Failure		500		{object}	happydns.Error
//	@Router			/users/{userId}/avatar.png [get]
func getUserAvatar(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	sizequery := c.DefaultQuery("size", "300")
	size, err := strconv.ParseInt(sizequery, 10, 32)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid size asked: %s", err.Error())})
		return
	} else if size > 2048 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Size too large."})
		return
	}

	ig, err := identicon.New(
		"happydomain", // namespace
		6,             // number of blocks (size)
		3,             // density of points
	)
	if err != nil {
		log.Printf("Unable to generate user avatar (uid=%s,user=%s): %s", user.Id.String(), user.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Unable to generate avatar."})
		return
	}

	ii, err := ig.Draw(user.Email)
	if err != nil {
		log.Printf("Unable to generate user avatar (uid=%s,user=%s): %s", user.Id.String(), user.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Unable to generate avatar."})
		return
	}

	c.Writer.Header().Set("Content-Type", "image/png")
	c.Writer.WriteHeader(http.StatusOK)
	ii.Png(int(size), c.Writer)
}

// getUserSettings gets the settings of the given user.
//
//	@Summary	Retrieve user's settings.
//	@Schemes
//	@Description	Retrieve the user's settings.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path	string	true	"User identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.UserSettings	"User settings"
//	@Failure		401	{object}	happydns.Error			"Authentication failure"
//	@Failure		403	{object}	happydns.Error			"Not your account"
//	@Router			/users/{userId}/settings [get]
func getUserSettings(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user.Settings)
}

// changeUserSettings updates the settings of the given user.
//
//	@Summary	Update user's settings.
//	@Schemes
//	@Description	Update the user's settings.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path	string					true	"User identifier"
//	@Param			body	body	happydns.UserSettings	true	"User settings"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.UserSettings	"User settings"
//	@Failure		400	{object}	happydns.Error			"Invalid input"
//	@Failure		401	{object}	happydns.Error			"Authentication failure"
//	@Failure		403	{object}	happydns.Error			"Not your account"
//	@Failure		500	{object}	happydns.Error
//	@Router			/users/{userId}/settings [post]
func changeUserSettings(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	var us happydns.UserSettings
	if err := c.ShouldBindJSON(&us); err != nil {
		log.Printf("%s sends invalid UserSettings JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	user.Settings = us

	if err := storage.MainStore.UpdateUser(user); err != nil {
		log.Printf("%s: unable to UpdateUser in changeUserSettings: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, user.Settings)
}

type passwordForm struct {
	Current         string
	Password        string
	PasswordConfirm string
}

// changePassword changes the password of the given account.
//
//	@Summary	Change password
//	@Schemes
//	@Description	Change the password of the given account.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			userId	path		string			true	"User identifier"
//	@Param			body	body		passwordForm	true	"Password confirmation"
//	@Success		204		{null}		null
//	@Failure		400		{object}	happydns.Error	"Invalid input"
//	@Failure		401		{object}	happydns.Error	"Authentication failure"
//	@Failure		403		{object}	happydns.Error	"Bad current password"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users/{userId}/new_password [post]
func changePassword(opts *config.Options, c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var lf passwordForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid passwordForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if !user.CheckAuth(lf.Current) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "The given current password is invalid."})
		return
	}

	if lf.Password != lf.PasswordConfirm {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "The new password and its confirmation are different."})
		return
	}

	if err := user.DefinePassword(lf.Password); err != nil {
		log.Printf("%s: unable to DefinePassword in changePassword: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	// Retrieve all user's sessions to disconnect them
	sessions, err := storage.MainStore.GetAuthUserSessions(user)
	if err != nil {
		log.Printf("%s: unable to GetUserSessions in changePassword: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	if err = storage.MainStore.UpdateAuthUser(user); err != nil {
		log.Printf("%s: unable to DefinePassword in changePassword: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	log.Printf("%s changes password for user %s", c.ClientIP(), user.Email)

	for _, session := range sessions {
		err = storage.MainStore.DeleteSession(session.Id)
		if err != nil {
			log.Printf("%s: unable to delete session (password changed): %s", c.ClientIP(), err.Error())
		}
	}

	logout(opts, c)
}

func deleteMyUser(opts *config.Options, c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	// Disallow route if user is authenticated through local service
	if _, err := storage.MainStore.GetAuthUser(user.Id); err == nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "This route is for external account only. Please use the route ./delete instead."})
		return
	}

	// Retrieve all user's sessions to disconnect them
	sessions, err := storage.MainStore.GetUserSessions(user)
	if err != nil {
		log.Printf("%s: unable to GetUserSessions in deleteUser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to delete your profile. Please try again later."})
		return
	}

	err = storage.MainStore.DeleteUser(user)
	if err != nil {
		log.Printf("%s: unable to DeleteUser in deletemyuser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	log.Printf("%s: deletes user: %s", c.ClientIP(), user.Email)

	for _, session := range sessions {
		err = storage.MainStore.DeleteSession(session.Id)
		if err != nil {
			log.Printf("%s: unable to delete session (drop account): %s", c.ClientIP(), err.Error())
		}
	}

	logout(opts, c)
}

// deleteUser delete the account related to the given user.
//
//	@Summary	Drop account
//	@Schemes
//	@Description	Delete the account related to the given user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			userId	path		string			true	"User identifier"
//	@Param			body	body		passwordForm	true	"Password confirmation"
//	@Success		204		{null}		null
//	@Failure		400		{object}	happydns.Error	"Invalid input"
//	@Failure		401		{object}	happydns.Error	"Authentication failure"
//	@Failure		403		{object}	happydns.Error	"Bad current password"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users/{userId}/delete [post]
func deleteUser(opts *config.Options, c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var lf passwordForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid passwordForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if !user.CheckAuth(lf.Current) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "The given current password is invalid."})
		return
	}

	// Retrieve all user's sessions to disconnect them
	sessions, err := storage.MainStore.GetAuthUserSessions(user)
	if err != nil {
		log.Printf("%s: unable to GetUserSessions in deleteUser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to delete your profile. Please try again later."})
		return
	}

	if err = storage.MainStore.DeleteAuthUser(user); err != nil {
		log.Printf("%s: unable to DeleteAuthUser in deleteuser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to delete your profile. Please try again later."})
		return
	}

	log.Printf("%s: deletes user: %s", c.ClientIP(), user.Email)

	for _, session := range sessions {
		err = storage.MainStore.DeleteSession(session.Id)
		if err != nil {
			log.Printf("%s: unable to delete session (drop account): %s", c.ClientIP(), err.Error())
		}
	}

	logout(opts, c)
}

func UserHandlerBase(c *gin.Context) (*happydns.User, error) {
	uid, err := base64.RawURLEncoding.DecodeString(c.Param("uid"))
	if err != nil {
		return nil, fmt.Errorf("Invalid user identifier given: %w", err)
	}

	user, err := storage.MainStore.GetUser(uid)
	if err != nil {
		return nil, fmt.Errorf("User not found")
	}

	return user, nil
}

func userHandler(c *gin.Context) {
	user, err := UserHandlerBase(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	c.Set("user", user)

	c.Next()
}

func UserAuthHandlerBase(c *gin.Context) (*happydns.UserAuth, error) {
	uid, err := base64.RawURLEncoding.DecodeString(c.Param("uid"))
	if err != nil {
		return nil, fmt.Errorf("Invalid user identifier given: %w", err)
	}

	user, err := storage.MainStore.GetAuthUser(uid)
	if err != nil {
		return nil, fmt.Errorf("User not found")
	}

	return user, nil
}

func userAuthHandler(c *gin.Context) {
	user, err := UserAuthHandlerBase(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	c.Set("authuser", user)

	c.Next()
}

type UploadedAddressValidation struct {
	// Key able to validate the email address.
	Key string
}

// validateUserAddress validates a user address after registration.
//
//	@Summary	Validate e-mail address.
//	@Schemes
//	@Description	This is the route called by the web interface in order to validate the e-mail address of the user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path		string						true	"User identifier"
//	@Param			body	body		UploadedAddressValidation	true	"Validation form"
//	@Success		204		{null}		null						"Email validated, you can now login"
//	@Failure		400		{object}	happydns.Error				"Invalid input"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users/{userId}/email [post]
func validateUserAddress(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var uav UploadedAddressValidation
	err := c.ShouldBindJSON(&uav)
	if err != nil {
		log.Printf("%s sends invalid AddressValidation JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := user.ValidateEmail(uav.Key); err != nil {
		log.Printf("%s bad email validation key: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Bad validation key: %s", err.Error())})
		return
	}

	if err := storage.MainStore.UpdateAuthUser(user); err != nil {
		log.Printf("%s: unable to UpdateUser in ValidateUserAddress: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	c.Status(http.StatusNoContent)
}

type UploadedAccountRecovery struct {
	// Key is the secret sent by email to the user.
	Key string

	// Password is the new password to use with this account.
	Password string
}

// recoverUserAccount performs account recovery by reseting the password of the account.
//
//	@Summary	Reset password with link in email.
//	@Schemes
//	@Description	This performs	account	recovery	by reseting the	password of the	account.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path		string					true	"User identifier"
//	@Param			body	body		UploadedAccountRecovery	true	"Recovery form"
//	@Success		204		{null}		null					"Recovery completed, you can now login with your new credentials"
//	@Failure		400		{object}	happydns.Error			"Invalid input"
//	@Failure		500		{object}	happydns.Error
//	@Router			/users/{userId}/recovery [post]
func recoverUserAccount(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var uar UploadedAccountRecovery
	err := c.ShouldBindJSON(&uar)
	if err != nil {
		log.Printf("%s sends invalid AccountRecovey JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := user.CanRecoverAccount(uar.Key); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": err.Error()})
		return
	}

	if len(uar.Password) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Password can't be empty!"})
		return
	}

	if err := user.DefinePassword(uar.Password); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	if err := storage.MainStore.UpdateAuthUser(user); err != nil {
		log.Printf("%s: unable to UpdateUser in recoverUserAccount: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	log.Printf("%s: User recovered: %s", c.ClientIP(), user.Email)
	c.Status(http.StatusNoContent)
}

func sessionHandler(c *gin.Context) {
	session, err := storage.MainStore.GetSession(c.Param("sid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	myuser := c.MustGet("LoggedUser").(*happydns.User)
	if !myuser.Id.Equals(session.IdUser) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "The session is not affiliated witht this user"})
		return
	}

	c.Set("session", session)

	c.Next()
}

// getSession gets the content of the current user's session.
//
//	@Summary	Retrieve user's session content
//	@Schemes
//	@Description	Get the content of the current user's session.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/session [get]
func getSession(c *gin.Context) {
	session := sessions.Default(c)

	s, err := storage.MainStore.GetSession(session.ID())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, s)
}

// clearSession removes the content of the current user's session.
//
//	@Summary	Remove user's session content
//	@Schemes
//	@Description	Remove the content of the current user's session.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		204	{null}		null
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/session [delete]
func clearSession(c *gin.Context) {
	session := sessions.Default(c)

	session.Clear()

	c.Status(http.StatusNoContent)
}

// clearUserSessions closes all existing sessions for a given user, and disconnect it.
//
//	@Summary	Remove user's sessions
//	@Schemes
//	@Description	Closes all sessions for a given user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		204	{null}		null
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/sessions [delete]
func clearUserSessions(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)

	sessions, err := storage.MainStore.GetUserSessions(myuser)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	for _, session := range sessions {
		err = storage.MainStore.DeleteSession(session.Id)
		if err != nil {
			log.Printf("Unable to DeleteSession(sid=%s) in clearUsersSessions(uid=%s): %s", session.Id, myuser.Id.String(), err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Unable to delete all sessions. Please try again."})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// getSessions lists the sessions open for the current user.
//
//	@Summary	List user's sessions
//	@Schemes
//	@Description	List the sessions open for the current user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/sessions [get]
func getSessions(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)

	s, err := storage.MainStore.GetUserSessions(myuser)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, s)
}

// createSession create a new session for the current user
//
//	@Summary	Create a new session for the current user.
//	@Schemes
//	@Description	Create a new session for the current user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/sessions [post]
func createSession(c *gin.Context) {
	var us happydns.Session
	err := c.ShouldBindJSON(&us)
	if err != nil {
		log.Printf("%s sends invalid Session JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	myuser := c.MustGet("LoggedUser").(*happydns.User)

	sessid := session.NewSessionId()
	mysession := &happydns.Session{
		Id:          sessid,
		IdUser:      myuser.Id,
		Description: us.Description,
		IssuedAt:    time.Now(),
		ExpiresOn:   time.Now().Add(24 * 365 * time.Hour),
	}

	err = storage.MainStore.UpdateSession(mysession)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": sessid})
}

// updateSession update a session owned by the current user
//
//	@Summary	Update a session owned by the current user.
//	@Schemes
//	@Description	Update	a session owned	by the current user.
//	@Tags			users
//	@Accept			json
//	@Param			sessionId	path	string	true	"Session identifier"
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/sessions/{sessionId} [put]
func updateSession(c *gin.Context) {
	var us happydns.Session
	err := c.ShouldBindJSON(&us)
	if err != nil {
		log.Printf("%s sends invalid Session JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	myuser := c.MustGet("LoggedUser").(*happydns.User)

	s, err := storage.MainStore.GetSession(c.Param("sid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	if !myuser.Id.Equals(s.IdUser) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "You are not allowed to update this session."})
		return
	}

	s.Description = us.Description
	s.ExpiresOn = us.ExpiresOn

	err = storage.MainStore.UpdateSession(s)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, s)
}

// deleteSession delete a session owned by the current user
//
//	@Summary	Delete a session owned by the current user.
//	@Schemes
//	@Description	Delete	a session owned	by the current user.
//	@Tags			users
//	@Accept			json
//	@Param			sessionId	path		string					true	"Session identifier"
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Session
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/sessions/{sessionId} [delete]
func deleteSession(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)

	s, err := storage.MainStore.GetSession(c.Param("sid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	if !myuser.Id.Equals(s.IdUser) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "You are not allowed to drop this session."})
		return
	}

	err = storage.MainStore.DeleteSession(c.Param("sid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, nil)
}
