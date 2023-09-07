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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

const NO_AUTH_ACCOUNT = "_no_auth"

func declareAuthenticationRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.POST("/auth", func(c *gin.Context) {
		checkAuth(opts, c)
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
	// Id is the user identifier
	Id happydns.Identifier `json:"id" swaggertype:"string"`

	// Email is the user email.
	Email string `json:"email"`

	// CreatedAt stores the date of the account creation.
	CreatedAt time.Time `json:"created_at,omitempty"`

	// Settings holds the user configuration.
	Settings happydns.UserSettings `json:"settings,omitempty"`
}

func currentUser(u *happydns.User) *DisplayUser {
	return &DisplayUser{
		Id:        u.Id,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		Settings:  u.Settings,
	}
}

// displayAuthToken returns the user information.
//
//	@Summary	User info.
//	@Schemes
//	@Description	Retrieve information about the currently logged user.
//	@Tags			user_auth
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	DisplayUser
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/auth [get]
func displayAuthToken(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	c.JSON(http.StatusOK, currentUser(user))
}

func displayNotAuthToken(opts *config.Options, c *gin.Context) *UserClaims {
	if !opts.NoAuth {
		requireLogin(opts, c, "Authorization required")
		return nil
	}

	claims, err := completeAuth(opts, c, UserProfile{
		UserId:        []byte{0},
		Email:         NO_AUTH_ACCOUNT,
		EmailVerified: true,
	})
	if err != nil {
		log.Printf("%s %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Something went wrong during your authentication. Please retry in a few minutes"})
		return nil
	}

	realUser, err := retrieveUserFromClaims(claims)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"errmsg": "Login success"})
	} else {
		c.JSON(http.StatusOK, currentUser(realUser))
	}

	return claims
}

// logout closes the user session.
//
//	@Summary	Close session.
//	@Schemes
//	@Description	Erase the HTTP-only cookie. This leads to user logout in its browser.
//	@Tags			user_auth
//	@Accept			json
//	@Produce		json
//	@Success		204	{null}		null			"Loged out"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/auth/logout [post]
func logout(opts *config.Options, c *gin.Context) {
	c.SetCookie(
		COOKIE_NAME,
		"",
		-1,
		opts.BaseURL+"/",
		"",
		opts.DevProxy == "" && opts.ExternalURL.URL.Scheme != "http",
		true,
	)
	c.Status(http.StatusNoContent)
}

type LoginForm struct {
	// Email of the user.
	Email string

	// Password of the user.
	Password string
}

// checkAuth validate user authentication and delivers a session token.
//
//	@Summary	Authenticate user.
//	@Schemes
//	@Description	Validate user authentication and delivers a session token.
//	@Tags			user_auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginForm		true	"Login information"
//	@Success		200		{object}	DisplayUser		"Login success"
//	@Failure		401		{object}	happydns.Error	"Authentication failure"
//	@Failure		500		{object}	happydns.Error
//	@Router			/auth [post]
func checkAuth(opts *config.Options, c *gin.Context) {
	var lf LoginForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid LoginForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	user, err := storage.MainStore.GetAuthUserByEmail(lf.Email)
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

	if user.EmailVerification == nil {
		log.Printf("%s tries to login as %q, but sent an invalid password", c.ClientIP(), lf.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Please validate your e-mail address before your first login.", "href": "/email-validation"})
		return
	}

	claims, err := completeAuth(opts, c, UserProfile{
		UserId:        user.Id,
		Email:         user.Email,
		EmailVerified: user.EmailVerification != nil,
		CreatedAt:     user.CreatedAt,
	})
	if err != nil {
		log.Printf("%s %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Something went wrong during your authentication. Please retry in a few minutes"})
		return
	}

	log.Printf("%s now logged as %q\n", c.ClientIP(), user.Email)

	realUser, err := retrieveUserFromClaims(claims)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"errmsg": "Login success"})
	} else {
		c.JSON(http.StatusOK, currentUser(realUser))
	}
}

func completeAuth(opts *config.Options, c *gin.Context, userprofile UserProfile) (*UserClaims, error) {
	// Issue a new JWT token
	jti := make([]byte, 16)
	_, err := rand.Read(jti)
	if err != nil {
		return nil, fmt.Errorf("unable to read enough random bytes: %w", err)
	}

	iat := jwt.NewNumericDate(time.Now())
	claims := &UserClaims{
		userprofile,
		jwt.RegisteredClaims{
			IssuedAt: iat,
			ID:       base64.StdEncoding.EncodeToString(jti),
		},
	}
	jwtToken := jwt.NewWithClaims(signingMethod, claims)
	jwtToken.Header["kid"] = "1"

	token, err := jwtToken.SignedString([]byte(opts.JWTSecretKey))
	if err != nil {
		return nil, fmt.Errorf("unable to sign user claims: %w", err)
	}

	c.SetCookie(
		COOKIE_NAME,      // name
		token,            // value
		30*24*3600,       // maxAge
		opts.BaseURL+"/", // path
		"",               // domain
		opts.DevProxy == "" && opts.ExternalURL.URL.Scheme != "http", // secure
		true, // httpOnly
	)

	return claims, nil
}
