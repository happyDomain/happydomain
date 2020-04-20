package api

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.POST("/api/users", apiHandler(registerUser))
}

type UploadedUser struct {
	Email    string
	Password string
}

func registerUser(opts *config.Options, p httprouter.Params, body io.Reader) Response {
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

	if user, err := happydns.NewUser(uu.Email, uu.Password); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if err := storage.MainStore.CreateUser(user); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: user,
		}
	}
}
