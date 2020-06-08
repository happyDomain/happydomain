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
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
	"git.happydns.org/happydns/utils"
)

func init() {
	router.POST("/api/users", ApiHandler(registerUser))
	router.PATCH("/api/users", ApiHandler(specialUserOperations))
	router.GET("/api/users/:uid", apiAuthHandler(sameUserHandler(getUser)))
	router.POST("/api/users/:uid/email", ApiHandler(userHandler(validateUserAddress)))
	router.POST("/api/users/:uid/recovery", ApiHandler(userHandler(recoverUserAccount)))
}

type UploadedUser struct {
	Kind     string
	Email    string
	Password string
}

func genUsername(user *happydns.User) (toName string) {
	if n := strings.Index(user.Email, "+"); n > 0 {
		toName = user.Email[0:n]
	} else {
		toName = user.Email[0:strings.Index(user.Email, "@")]
	}
	if len(toName) > 1 {
		toNameCopy := strings.Replace(toName, ".", " ", -1)
		toName = ""
		lastRuneIsSpace := true
		for _, runeValue := range toNameCopy {
			if lastRuneIsSpace {
				lastRuneIsSpace = false
				toName += string(unicode.ToTitle(runeValue))
			} else {
				toName += string(runeValue)
			}

			if unicode.IsSpace(runeValue) || unicode.IsPunct(runeValue) || unicode.IsSymbol(runeValue) {
				lastRuneIsSpace = true
			}
		}
	}
	return
}

func sendValidationLink(opts *config.Options, user *happydns.User) error {
	toName := genUsername(user)
	return utils.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Your new account on happyDNS",
		`Welcome to happyDNS!
--------------------

Hi `+toName+`,

We are pleased that you created an account on our great domain name
management platform!

In order to validate your account, please follow this link now:

[Validate my account](`+opts.GetRegistrationURL(user)+`)`,
	)
}

func sendRecoveryLink(opts *config.Options, user *happydns.User) error {
	toName := genUsername(user)
	return utils.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Recover you happyDNS account",
		`Hi `+toName+`,

You've just ask on our platform to recover your account.

In order to define a new password, please follow this link now:

[Recover my account](`+opts.GetAccountRecoveryURL(user)+`)`,
	)
}

func registerUser(opts *config.Options, p httprouter.Params, body io.Reader) Response {
	var uu UploadedUser
	err := json.NewDecoder(body).Decode(&uu)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	if len(uu.Email) <= 3 || strings.Index(uu.Email, "@") == -1 {
		return APIErrorResponse{
			err: errors.New("The given email is invalid."),
		}
	}

	if len(uu.Password) <= 7 {
		return APIErrorResponse{
			err: errors.New("The given email is invalid."),
		}
	}

	if storage.MainStore.UserExists(uu.Email) {
		return APIErrorResponse{
			err: errors.New("An account already exists with the given address. Try login now."),
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
	} else if sendValidationLink(opts, user); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: user,
		}
	}
}

func specialUserOperations(opts *config.Options, p httprouter.Params, body io.Reader) Response {
	var uu UploadedUser
	err := json.NewDecoder(body).Decode(&uu)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	res := APIErrorResponse{
		err:    errors.New("If this address exists in our database, you'll receive a new e-mail."),
		status: http.StatusOK,
	}

	if user, err := storage.MainStore.GetUserByEmail(uu.Email); err != nil {
		return res
	} else {
		if uu.Kind == "recovery" {
			if user.EmailValidated == nil {
				if err = sendValidationLink(opts, user); err != nil {
					return APIErrorResponse{
						err: err,
					}

				}
			} else {
				if err = sendRecoveryLink(opts, user); err != nil {
					return APIErrorResponse{
						err: err,
					}

				} else if err := storage.MainStore.UpdateUser(user); err != nil {
					return APIErrorResponse{
						err: fmt.Errorf("An error occurs when trying to recover your account: %w", err),
					}
				}
			}
		} else if uu.Kind == "validation" {
			if user.EmailValidated != nil {
				return res
			} else if err = sendValidationLink(opts, user); err != nil {
				return APIErrorResponse{
					err: err,
				}
			}
		}
	}

	return res
}

func sameUserHandler(f func(*config.Options, *happydns.User, io.Reader) Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, u *happydns.User, ps httprouter.Params, body io.Reader) Response {
		if uid, err := strconv.ParseInt(ps.ByName("uid"), 16, 64); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    fmt.Errorf("Invalid user identifier given: %w", err),
			}
		} else if user, err := storage.MainStore.GetUser(uid); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("User not found"),
			}
		} else if user.Id != u.Id {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("User not found"),
			}
		} else {
			return f(opts, user, body)
		}
	}
}

func getUser(opts *config.Options, user *happydns.User, _ io.Reader) Response {
	return APIResponse{
		response: user,
	}
}

func userHandler(f func(*config.Options, *happydns.User, io.Reader) Response) func(*config.Options, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, ps httprouter.Params, body io.Reader) Response {
		if uid, err := strconv.ParseInt(ps.ByName("uid"), 16, 64); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    fmt.Errorf("Invalid user identifier given: %w", err),
			}
		} else if user, err := storage.MainStore.GetUser(uid); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("User not found"),
			}
		} else {
			return f(opts, user, body)
		}
	}
}

type UploadedAddressValidation struct {
	Key string
}

func validateUserAddress(opts *config.Options, user *happydns.User, body io.Reader) Response {
	var uav UploadedAddressValidation
	err := json.NewDecoder(body).Decode(&uav)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	if err := user.ValidateEmail(uav.Key); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if err := storage.MainStore.UpdateUser(user); err != nil {
		return APIErrorResponse{
			status: http.StatusNotFound,
			err:    errors.New("User not found"),
		}
	} else {
		return APIResponse{
			response: true,
		}
	}
}

type UploadedAccountRecovery struct {
	Key      string
	Password string
}

func recoverUserAccount(opts *config.Options, user *happydns.User, body io.Reader) Response {
	var uar UploadedAccountRecovery
	err := json.NewDecoder(body).Decode(&uar)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	if err := user.CanRecoverAccount(uar.Key); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if len(uar.Password) == 0 {
		return APIResponse{
			response: false,
		}
	} else if err := user.DefinePassword(uar.Password); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if err := storage.MainStore.UpdateUser(user); err != nil {
		return APIErrorResponse{
			status: http.StatusNotFound,
			err:    errors.New("User not found"),
		}
	} else {
		return APIResponse{
			response: true,
		}
	}
}
