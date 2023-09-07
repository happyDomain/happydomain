// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/actions"
	"git.happydns.org/happyDomain/config"
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

	apiUserRoutes := router.Group("/users/:uid")
	apiUserRoutes.Use(userHandler)
	apiUserRoutes.Use(SameUserHandler)

	apiUserRoutes.GET("", getUser)
	apiUserRoutes.GET("/settings", getUserSettings)
	apiUserRoutes.POST("/settings", changeUserSettings)

	apiUserAuthRoutes := router.Group("/users/:uid")
	apiUserAuthRoutes.Use(userAuthHandler)
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

func getUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user)
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
		err = storage.MainStore.DeleteSession(session)
		if err != nil {
			log.Printf("%s: unable to delete session (password changed): %s", c.ClientIP(), err.Error())
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	if err = storage.MainStore.DeleteAuthUser(user); err != nil {
		log.Printf("%s: unable to DefinePassword in deleteuser: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	log.Printf("%s: deletes user: %s", c.ClientIP(), user.Email)

	for _, session := range sessions {
		err = storage.MainStore.DeleteSession(session)
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
//	@Router			/sessions [get]
func getSession(c *gin.Context) {
	session := c.MustGet("MySession").(*happydns.Session)

	c.JSON(http.StatusOK, session)
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
//	@Router			/sessions [delete]
func clearSession(c *gin.Context) {
	session := c.MustGet("MySession").(*happydns.Session)

	session.ClearSession()

	c.Status(http.StatusNoContent)
}
