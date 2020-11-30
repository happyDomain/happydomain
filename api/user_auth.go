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
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

var AuthFunc = checkAuth

func init() {
	router.GET("/api/auth", apiAuthHandler(displayAuthToken))
	router.POST("/api/auth", ApiHandler(func(opts *config.Options, ps httprouter.Params, b io.Reader) Response {
		return AuthFunc(opts, ps, b)
	}))
	router.POST("/api/auth/logout", ApiHandler(logout))
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

func displayAuthToken(_ *config.Options, req *RequestResources, _ io.Reader) Response {
	return APIResponse{
		response: currentUser(req.User),
	}
}

func completeAuth(opts *config.Options, email string, service string) Response {
	var usr *happydns.User
	var err error

	if !storage.MainStore.UserExists(email) {
		now := time.Now()
		usr = &happydns.User{
			Email:            email,
			RegistrationTime: &now,
		}
		if err = storage.MainStore.CreateUser(usr); err != nil {
			return APIErrorResponse{
				err: err,
			}
		}
		log.Printf("Create new user after successful service=%q login %q\n", service, usr)
	} else if usr, err = storage.MainStore.GetUserByEmail(email); err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	log.Printf("New user logged as %q\n", usr.Email)

	var session *happydns.Session
	if session, err = happydns.NewSession(usr); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if err = storage.MainStore.CreateSession(session); err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: currentUser(usr),
		cookies: []*http.Cookie{&http.Cookie{
			Name:     "happydns_session",
			Value:    base64.StdEncoding.EncodeToString(session.Id),
			Path:     opts.BaseURL + "/",
			Expires:  time.Now().Add(30 * 24 * time.Hour),
			Secure:   opts.DevProxy == "",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		}},
	}
}

func logout(opts *config.Options, _ httprouter.Params, body io.Reader) Response {
	return APIResponse{
		response: true,
		cookies: []*http.Cookie{&http.Cookie{
			Name:     "happydns_session",
			Value:    "",
			Path:     opts.BaseURL + "/",
			Expires:  time.Unix(0, 0),
			Secure:   opts.DevProxy == "",
			HttpOnly: true,
		}},
	}
}

type loginForm struct {
	Email    string
	Password string
}

func dummyAuth(opts *config.Options, _ httprouter.Params, body io.Reader) Response {
	var lf loginForm
	if err := json.NewDecoder(body).Decode(&lf); err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if user, err := storage.MainStore.GetUserByEmail(lf.Email); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return completeAuth(opts, user.Email, "dummy")
	}
}

func checkAuth(opts *config.Options, _ httprouter.Params, body io.Reader) Response {
	var lf loginForm
	if err := json.NewDecoder(body).Decode(&lf); err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if user, err := storage.MainStore.GetUserByEmail(lf.Email); err != nil {
		return APIErrorResponse{
			err:    errors.New(`Invalid username or password.`),
			status: http.StatusUnauthorized,
		}
	} else if !user.CheckAuth(lf.Password) {
		return APIErrorResponse{
			err:    errors.New(`Invalid username or password.`),
			status: http.StatusUnauthorized,
		}
	} else if user.EmailValidated == nil {
		return APIErrorResponse{
			err:    errors.New(`Please validate your e-mail address before your first login.`),
			href:   "/email-validation",
			status: http.StatusUnauthorized,
		}
	} else {
		return completeAuth(opts, user.Email, "local")
	}
}
