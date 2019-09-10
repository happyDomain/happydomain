package api

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/julienschmidt/httprouter"

	"git.nemunai.re/libredns/struct"
)

func init() {
	router.GET("/api/users", apiHandler(listUsers))
	router.POST("/api/users", apiHandler(registerUser))
}

func listUsers(_ httprouter.Params, _ io.Reader) Response {
	if users, err := libredns.GetUsers(); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: users,
		}
	}
}

type UploadedUser struct {
	Email            string
	Password         string
}

func registerUser(p httprouter.Params, body io.Reader) Response {
	var uu UploadedUser
	err := json.NewDecoder(body).Decode(&uu)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if len(uu.Email) <= 3 {
		return APIErrorResponse{
			err: errors.New("The given email is invalid."),
		}
	}

	if user, err := libredns.NewUser(uu.Email, uu.Password); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: user,
		}
	}
}
