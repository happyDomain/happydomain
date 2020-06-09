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

package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
	"git.happydns.org/happydns/utils"
)

func init() {
	router.GET("/api/users", api.ApiHandler(getUsers))
	router.POST("/api/users", api.ApiHandler(newUser))
	router.DELETE("/api/users", api.ApiHandler(deleteUsers))

	router.GET("/api/users/:userid", api.ApiHandler(userHandler(getUser)))
	router.PUT("/api/users/:userid", api.ApiHandler(userHandler(updateUser)))
	router.DELETE("/api/users/:userid", api.ApiHandler(userHandler(deleteUser)))

	router.POST("/api/users/:userid/recover_link", api.ApiHandler(userHandler(recoverUserAcct)))
	router.POST("/api/users/:userid/reset_password", api.ApiHandler(userHandler(resetUserPasswd)))
	router.POST("/api/users/:userid/send_recover_email", api.ApiHandler(userHandler(sendRecoverUserAcct)))
	router.POST("/api/users/:userid/send_validation_email", api.ApiHandler(userHandler(sendValidateUserEmail)))
	router.POST("/api/users/:userid/validation_link", api.ApiHandler(userHandler(emailValidationLink)))
	router.POST("/api/users/:userid/validate_email", api.ApiHandler(userHandler(validateEmail)))
}

func getUsers(_ *config.Options, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(storage.MainStore.GetUsers())
}

func newUser(_ *config.Options, _ httprouter.Params, body io.Reader) api.Response {
	uu := &happydns.User{}
	err := json.NewDecoder(body).Decode(&uu)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}
	uu.Id = 0

	return api.NewAPIResponse(uu, storage.MainStore.CreateUser(uu))
}

func deleteUsers(_ *config.Options, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(true, storage.MainStore.ClearUsers())
}

func userHandler(f func(*config.Options, *happydns.User, httprouter.Params, io.Reader) api.Response) func(*config.Options, httprouter.Params, io.Reader) api.Response {
	return func(opts *config.Options, ps httprouter.Params, body io.Reader) api.Response {
		userid, err := strconv.ParseInt(ps.ByName("userid"), 10, 64)
		if err != nil {
			user, err := storage.MainStore.GetUserByEmail(ps.ByName("userid"))
			if err != nil {
				return api.NewAPIErrorResponse(http.StatusNotFound, err)
			} else {
				return f(opts, user, ps, body)
			}
		} else {
			user, err := storage.MainStore.GetUser(userid)
			if err != nil {
				return api.NewAPIErrorResponse(http.StatusNotFound, err)
			} else {
				return f(opts, user, ps, body)
			}
		}
	}
}

func getUser(_ *config.Options, user *happydns.User, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(user, nil)
}

func updateUser(_ *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	uu := &happydns.User{}
	err := json.NewDecoder(body).Decode(&uu)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}
	uu.Id = user.Id

	return api.NewAPIResponse(uu, storage.MainStore.UpdateUser(uu))
}

func deleteUser(_ *config.Options, user *happydns.User, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(true, storage.MainStore.DeleteUser(user))
}

func emailValidationLink(opts *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	return api.NewAPIResponse(opts.GetRegistrationURL(user), nil)
}

func recoverUserAcct(opts *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	return api.NewAPIResponse(opts.GetAccountRecoveryURL(user), nil)
}

type resetPassword struct {
	Password string
}

func resetUserPasswd(_ *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	urp := &resetPassword{}
	err := json.NewDecoder(body).Decode(&urp)
	if err != nil && err != io.EOF {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}

	if urp.Password == "" {
		urp.Password, err = utils.GeneratePassword()
		if err != nil {
			return api.NewAPIErrorResponse(http.StatusInternalServerError, err)
		}
	}

	err = user.DefinePassword(urp.Password)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusInternalServerError, err)
	}

	return api.NewAPIResponse(urp, storage.MainStore.UpdateUser(user))
}

func sendRecoverUserAcct(opts *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	return api.NewAPIResponse(true, api.SendRecoveryLink(opts, user))
}

func sendValidateUserEmail(opts *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	return api.NewAPIResponse(true, api.SendValidationLink(opts, user))
}

func validateEmail(_ *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	now := time.Now()
	user.EmailValidated = &now
	return api.NewAPIResponse(user, storage.MainStore.UpdateUser(user))
}
