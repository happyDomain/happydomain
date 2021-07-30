// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

const NO_AUTH_ACCOUNT = "_no_auth"

var AuthFunc = checkAuth

func declareAuthenticationRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.POST("/auth", func(c *gin.Context) {
		AuthFunc(opts, c)
	})
	router.POST("/auth/logout", func(c *gin.Context) {
		logout(opts, c)
	})

	apiAuthRoutes := router.Group("/auth")
	apiAuthRoutes.Use(authMiddleware(opts, true))

	apiAuthRoutes.GET("", func(c *gin.Context) {
		if _, exists := c.Get("MySession"); exists {
			displayAuthToken(c)
		} else {
			displayNotAuthToken(opts, c)
		}
	})
}

type DisplayUser struct {
	Id               int64                 `json:"id"`
	Email            string                `json:"email"`
	RegistrationTime *time.Time            `json:"registration_time,omitempty"`
	Settings         happydns.UserSettings `json:"settings,omitempty"`
}

func currentUser(u *happydns.User) *DisplayUser {
	return &DisplayUser{
		Id:               u.Id,
		Email:            u.Email,
		RegistrationTime: u.RegistrationTime,
		Settings:         u.Settings,
	}
}

func displayNotAuthToken(opts *config.Options, c *gin.Context) {
	if !opts.NoAuth {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Authorization required"})
		return
	}

	completeAuth(opts, c, NO_AUTH_ACCOUNT, NO_AUTH_ACCOUNT)
}

func displayAuthToken(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	c.JSON(http.StatusOK, currentUser(user))
}

func completeAuth(opts *config.Options, c *gin.Context, email string, service string) {
	var usr *happydns.User
	var err error

	if !storage.MainStore.UserExists(email) {
		now := time.Now()
		usr = &happydns.User{
			Email:            email,
			RegistrationTime: &now,
		}
		if err = storage.MainStore.CreateUser(usr); err != nil {
			log.Printf("%s: unable to CreateUser in completeAuth: %s", c.ClientIP(), err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create your account. Please try again later."})
			return
		}
		log.Printf("%s: Creates new user after successful service=%q login %q\n", c.ClientIP(), service, usr)
	} else if usr, err = storage.MainStore.GetUserByEmail(email); err != nil {
		log.Printf("%s: unable to find User in completeAuth: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to access your account. Please try again later."})
		return
	}

	log.Printf("%s now logged as %q\n", c.ClientIP(), usr.Email)

	var session *happydns.Session
	if session, err = happydns.NewSession(usr); err != nil {
		log.Printf("%s: unable to NewSession in completeAuth: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create your session. Please try again later."})
		return
	} else if err = storage.MainStore.CreateSession(session); err != nil {
		log.Printf("%s: unable to CreateSession in completeAuth: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to create your session. Please try again later."})
		return
	}

	c.SetCookie(
		COOKIE_NAME, // name
		base64.StdEncoding.EncodeToString(session.Id), // value
		30*24*3600,       // maxAge
		opts.BaseURL+"/", // path
		"",               // domain
		opts.DevProxy == "" && !strings.HasPrefix(opts.ExternalURL, "http://"), // secure
		true, // httpOnly
	)

	c.JSON(http.StatusOK, currentUser(usr))
}

func logout(opts *config.Options, c *gin.Context) {
	c.SetCookie(
		COOKIE_NAME,
		"",
		-1,
		opts.BaseURL+"/",
		"",
		opts.DevProxy == "" && !strings.HasPrefix(opts.ExternalURL, "http://"),
		true,
	)
	c.JSON(http.StatusOK, true)
}

type loginForm struct {
	Email    string
	Password string
}

func dummyAuth(opts *config.Options, c *gin.Context) {
	var lf loginForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid LoginForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	user, err := storage.MainStore.GetUserByEmail(lf.Email)
	if err != nil {
		log.Printf("%s user's email (%s) not found: %s", c.ClientIP(), lf.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Invalid username or password."})
		return
	}

	completeAuth(opts, c, user.Email, "dummy")
}

func checkAuth(opts *config.Options, c *gin.Context) {
	var lf loginForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid LoginForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	user, err := storage.MainStore.GetUserByEmail(lf.Email)
	if err != nil {
		log.Printf("%s user's email (%s) not found: %s", c.ClientIP(), lf.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Invalid username or password."})
		return
	}

	if !user.CheckAuth(lf.Password) {
		log.Printf("%s tries to login as %q, but sent an invalid password", c.ClientIP(), lf.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Invalid username or password."})
		return
	}

	if user.EmailValidated == nil {
		log.Printf("%s tries to login as %q, but sent an invalid password", c.ClientIP(), lf.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Please validate your e-mail address before your first login.", "href": "/email-validation"})
		return
	}

	completeAuth(opts, c, user.Email, "local")
}
