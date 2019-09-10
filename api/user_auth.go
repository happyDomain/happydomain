package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"git.nemunai.re/libredns/struct"
)

var AuthFunc = checkAuth

func init() {
	router.GET("/api/users/auth", apiAuthHandler(validateAuthToken))
	router.POST("/api/users/auth", apiHandler(func(ps httprouter.Params, b io.Reader) (Response) {
		return AuthFunc(ps, b)
	}))
}

func validateAuthToken(u libredns.User, _ httprouter.Params, _ io.Reader) (Response) {
	return APIResponse{
		response: u,
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

	if user, err := libredns.GetUserByEmail(lf.Email); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		session, err := user.NewSession()
		if err != nil {
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

	if user, err := libredns.GetUserByEmail(lf.Email); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if !user.CheckAuth(lf.Password) {
		return APIErrorResponse{
			err: errors.New(`{"status": "Invalid username or password"}`),
			status: http.StatusUnauthorized,
		}
	} else {
		session, err := user.NewSession()
		if err != nil {
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
