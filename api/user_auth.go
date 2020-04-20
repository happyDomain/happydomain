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
