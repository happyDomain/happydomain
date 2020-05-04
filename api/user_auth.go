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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

var AuthFunc = checkAuth

func init() {
	router.GET("/api/users/auth", apiAuthHandler(validateAuthToken))
	router.POST("/api/users/auth", apiHandler(func(_ *config.Options, ps httprouter.Params, b io.Reader) Response {
		return AuthFunc(ps, b)
	}))
}

type DisplayUser struct {
	Id               int64      `json:"id"`
	Email            string     `json:"email"`
	RegistrationTime *time.Time `json:"registration_time,omitempty"`
}

func validateAuthToken(_ *config.Options, u *happydns.User, _ httprouter.Params, _ io.Reader) Response {
	return APIResponse{
		response: &DisplayUser{
			Id:               u.Id,
			Email:            u.Email,
			RegistrationTime: u.RegistrationTime,
		},
	}
}

type loginForm struct {
	Email    string
	Password string
}

func dummyAuth(_ httprouter.Params, body io.Reader) Response {
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
		session, err := happydns.NewSession(user)
		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		if err := storage.MainStore.CreateSession(session); err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		res := map[string]interface{}{}
		res["status"] = "OK"
		res["id_session"] = session.Id

		return APIResponse{
			response: res,
		}
	}
}

func checkAuth(_ httprouter.Params, body io.Reader) Response {
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
	} else if !user.CheckAuth(lf.Password) {
		return APIErrorResponse{
			err:    errors.New(`Invalid username or password`),
			status: http.StatusUnauthorized,
		}
	} else {
		session, err := happydns.NewSession(user)
		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		if err := storage.MainStore.CreateSession(session); err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		res := map[string]interface{}{}
		res["status"] = "OK"
		res["id_session"] = session.Id

		return APIResponse{
			response: res,
		}
	}
}
