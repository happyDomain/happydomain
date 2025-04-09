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

package happydns

const TryAgainErr = "Sorry, we are currently unable to sent email validation link. Please try again later."

type ErrorResponse struct {
	// Message describe the error to display to the user.
	Message string `json:"errmsg"`

	// Link is a link that can help the user to fix the error.
	Link string `json:"href,omitempty"`
}

type InternalError struct {
	Err         error
	UserMessage string
	UserLink    string
	HTTPStatus  int
}

func (err InternalError) Error() string {
	return err.Err.Error()
}

func (err InternalError) ToErrorResponse() ErrorResponse {
	if err.UserMessage == "" {
		return ErrorResponse{
			Message: err.Err.Error(),
			Link:    err.UserLink,
		}
	}

	return ErrorResponse{
		Message: err.UserMessage,
		Link:    err.UserLink,
	}
}
