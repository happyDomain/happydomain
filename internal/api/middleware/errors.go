// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// ErrorResponse answers err. An error carrying its own status and message, a
// happydns.InternalError or another happydns.HTTPError, is found however a
// use case wrapped it on the way, and answered with them: the wrapping text
// is left out, as it may hold what the user is not meant to see.
func ErrorResponse(c *gin.Context, defaultStatus int, err error) {
	var ie happydns.InternalError
	var e happydns.HTTPError
	if errors.As(err, &ie) {
		// The whole chain, for the logs only.
		log.Println(err.Error())

		status := ie.HTTPStatus()
		if status == 0 {
			status = http.StatusInternalServerError
		}

		c.AbortWithStatusJSON(status, ie.ToErrorResponse())
		return
	} else if errors.As(err, &e) {
		status := e.HTTPStatus()
		if status == 0 {
			status = http.StatusInternalServerError
		}

		c.AbortWithStatusJSON(status, e.ToErrorResponse())
		return
	} else if errors.Is(err, happydns.ErrAuthUserNotFound) || errors.Is(err, happydns.ErrDomainNotFound) || errors.Is(err, happydns.ErrDomainLogNotFound) || errors.Is(err, happydns.ErrProviderNotFound) || errors.Is(err, happydns.ErrSessionNotFound) || errors.Is(err, happydns.ErrUserNotFound) || errors.Is(err, happydns.ErrZoneNotFound) {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{
			Message: err.Error(),
		})
		return
	} else if errors.Is(err, happydns.ErrUserAlreadyExist) {
		c.AbortWithStatusJSON(http.StatusConflict, happydns.ErrorResponse{
			Message: err.Error(),
		})
		return
	}

	c.AbortWithStatusJSON(defaultStatus, happydns.ErrorResponse{
		Message: err.Error(),
	})
}
