// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package secret

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"git.happydns.org/happyDomain/model"
)

// userError is what a user is told about a stored secret that does not
// open: what to do, without the reason, which names safes. The reason stays
// reachable with errors.Is.
type userError struct {
	msg    string
	status int
	err    error
}

var _ happydns.HTTPError = userError{}

func (e userError) Error() string { return e.msg }
func (e userError) Unwrap() error { return e.err }
func (e userError) HTTPStatus() int {
	return e.status
}
func (e userError) ToErrorResponse() happydns.ErrorResponse {
	return happydns.ErrorResponse{Message: e.msg}
}

// UserError returns what to tell the user about err, met while opening or
// sealing the secrets of one of their objects, named by what ("provider",
// "notification channel"). It returns nil when err is not about a stored
// secret that does not open, so that callers handle the rest as before:
//
//	if uerr := secret.UserError(err, "provider"); uerr != nil {
//		return uerr
//	}
//
// It is the one place these errors are translated: call it wherever secrets
// are opened or sealed on a user's behalf. An error translated already is
// returned as it was, however wrapped since.
func UserError(err error, what string) error {
	var ue userError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ue):
		return ue
	case errors.Is(err, ErrSafeUnavailable):
		// The administrator's to repair: logged for them, as the user
		// only gets told to reach them.
		log.Printf("secret: %s", err)
		return userError{
			msg:    fmt.Sprintf("the stored credentials of this %s cannot be decrypted at the moment, because of the configuration of this instance: please contact its administrator", what),
			status: http.StatusServiceUnavailable,
			err:    err,
		}
	case errors.Is(err, ErrUnopenable):
		return userError{
			msg:    fmt.Sprintf("a stored credential of this %s can no longer be decrypted: enter it again", what),
			status: http.StatusBadRequest,
			err:    err,
		}
	}
	return nil
}
