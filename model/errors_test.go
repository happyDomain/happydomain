// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package happydns_test

import (
	"errors"
	"net/http"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestCustomErrorError(t *testing.T) {
	baseErr := errors.New("test error message")
	customErr := happydns.CustomError{
		Err:      baseErr,
		UserLink: "https://example.com/help",
		Status:   http.StatusBadRequest,
	}

	if customErr.Error() != "test error message" {
		t.Errorf("CustomError.Error() = %q; want %q", customErr.Error(), "test error message")
	}
}

func TestCustomErrorToErrorResponse(t *testing.T) {
	baseErr := errors.New("custom error")
	customErr := happydns.CustomError{
		Err:      baseErr,
		UserLink: "https://example.com/docs",
		Status:   http.StatusBadRequest,
	}

	resp := customErr.ToErrorResponse()

	if resp.Message != "custom error" {
		t.Errorf("ToErrorResponse().Message = %q; want %q", resp.Message, "custom error")
	}

	if resp.Link != "https://example.com/docs" {
		t.Errorf("ToErrorResponse().Link = %q; want %q", resp.Link, "https://example.com/docs")
	}
}

func TestCustomErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		expectedStatus int
	}{
		{
			name:           "bad request",
			status:         http.StatusBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found",
			status:         http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			status:         http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customErr := happydns.CustomError{
				Err:    errors.New("test"),
				Status: tt.status,
			}

			if customErr.HTTPStatus() != tt.expectedStatus {
				t.Errorf("HTTPStatus() = %d; want %d", customErr.HTTPStatus(), tt.expectedStatus)
			}
		})
	}
}

func TestForbiddenErrorError(t *testing.T) {
	forbiddenErr := happydns.ForbiddenError{
		Msg: "access denied",
	}

	if forbiddenErr.Error() != "access denied" {
		t.Errorf("ForbiddenError.Error() = %q; want %q", forbiddenErr.Error(), "access denied")
	}
}

func TestForbiddenErrorToErrorResponse(t *testing.T) {
	forbiddenErr := happydns.ForbiddenError{
		Msg: "forbidden resource",
	}

	resp := forbiddenErr.ToErrorResponse()

	if resp.Message != "forbidden resource" {
		t.Errorf("ToErrorResponse().Message = %q; want %q", resp.Message, "forbidden resource")
	}

	if resp.Link != "" {
		t.Errorf("ToErrorResponse().Link = %q; want empty string", resp.Link)
	}
}

func TestForbiddenErrorHTTPStatus(t *testing.T) {
	forbiddenErr := happydns.ForbiddenError{
		Msg: "test",
	}

	if forbiddenErr.HTTPStatus() != http.StatusForbidden {
		t.Errorf("HTTPStatus() = %d; want %d", forbiddenErr.HTTPStatus(), http.StatusForbidden)
	}
}

func TestInternalErrorError(t *testing.T) {
	baseErr := errors.New("internal error message")
	internalErr := happydns.InternalError{
		Err:         baseErr,
		UserMessage: "something went wrong",
		UserLink:    "https://example.com/status",
	}

	if internalErr.Error() != "internal error message" {
		t.Errorf("InternalError.Error() = %q; want %q", internalErr.Error(), "internal error message")
	}
}

func TestInternalErrorToErrorResponse(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		userMessage     string
		userLink        string
		expectedMessage string
		expectedLink    string
	}{
		{
			name:            "with user message",
			err:             errors.New("internal error"),
			userMessage:     "user-friendly message",
			userLink:        "https://example.com/help",
			expectedMessage: "user-friendly message",
			expectedLink:    "https://example.com/help",
		},
		{
			name:            "without user message",
			err:             errors.New("internal error"),
			userMessage:     "",
			userLink:        "https://example.com/help",
			expectedMessage: "internal error",
			expectedLink:    "https://example.com/help",
		},
		{
			name:            "no link",
			err:             errors.New("error"),
			userMessage:     "user message",
			userLink:        "",
			expectedMessage: "user message",
			expectedLink:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			internalErr := happydns.InternalError{
				Err:         tt.err,
				UserMessage: tt.userMessage,
				UserLink:    tt.userLink,
			}

			resp := internalErr.ToErrorResponse()

			if resp.Message != tt.expectedMessage {
				t.Errorf("ToErrorResponse().Message = %q; want %q", resp.Message, tt.expectedMessage)
			}

			if resp.Link != tt.expectedLink {
				t.Errorf("ToErrorResponse().Link = %q; want %q", resp.Link, tt.expectedLink)
			}
		})
	}
}

func TestInternalErrorHTTPStatus(t *testing.T) {
	internalErr := happydns.InternalError{
		Err: errors.New("test"),
	}

	if internalErr.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("HTTPStatus() = %d; want %d", internalErr.HTTPStatus(), http.StatusInternalServerError)
	}
}

func TestNotFoundErrorError(t *testing.T) {
	notFoundErr := happydns.NotFoundError{
		Msg: "resource not found",
	}

	if notFoundErr.Error() != "resource not found" {
		t.Errorf("NotFoundError.Error() = %q; want %q", notFoundErr.Error(), "resource not found")
	}
}

func TestNotFoundErrorToErrorResponse(t *testing.T) {
	notFoundErr := happydns.NotFoundError{
		Msg: "page not found",
	}

	resp := notFoundErr.ToErrorResponse()

	if resp.Message != "page not found" {
		t.Errorf("ToErrorResponse().Message = %q; want %q", resp.Message, "page not found")
	}

	if resp.Link != "" {
		t.Errorf("ToErrorResponse().Link = %q; want empty string", resp.Link)
	}
}

func TestNotFoundErrorHTTPStatus(t *testing.T) {
	notFoundErr := happydns.NotFoundError{
		Msg: "test",
	}

	if notFoundErr.HTTPStatus() != http.StatusNotFound {
		t.Errorf("HTTPStatus() = %d; want %d", notFoundErr.HTTPStatus(), http.StatusNotFound)
	}
}

func TestValidationErrorError(t *testing.T) {
	validationErr := happydns.ValidationError{
		Msg: "validation failed",
	}

	if validationErr.Error() != "validation failed" {
		t.Errorf("ValidationError.Error() = %q; want %q", validationErr.Error(), "validation failed")
	}
}

func TestValidationErrorToErrorResponse(t *testing.T) {
	validationErr := happydns.ValidationError{
		Msg: "invalid input",
	}

	resp := validationErr.ToErrorResponse()

	if resp.Message != "invalid input" {
		t.Errorf("ToErrorResponse().Message = %q; want %q", resp.Message, "invalid input")
	}

	if resp.Link != "" {
		t.Errorf("ToErrorResponse().Link = %q; want empty string", resp.Link)
	}
}

func TestValidationErrorHTTPStatus(t *testing.T) {
	validationErr := happydns.ValidationError{
		Msg: "test",
	}

	if validationErr.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("HTTPStatus() = %d; want %d", validationErr.HTTPStatus(), http.StatusBadRequest)
	}
}

func TestHTTPErrorInterface(t *testing.T) {
	tests := []struct {
		name     string
		err      happydns.HTTPError
		wantCode int
	}{
		{
			name: "CustomError",
			err: happydns.CustomError{
				Err:    errors.New("test"),
				Status: http.StatusTeapot,
			},
			wantCode: http.StatusTeapot,
		},
		{
			name: "ForbiddenError",
			err: happydns.ForbiddenError{
				Msg: "forbidden",
			},
			wantCode: http.StatusForbidden,
		},
		{
			name: "InternalError",
			err: happydns.InternalError{
				Err: errors.New("test"),
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "NotFoundError",
			err: happydns.NotFoundError{
				Msg: "not found",
			},
			wantCode: http.StatusNotFound,
		},
		{
			name: "ValidationError",
			err: happydns.ValidationError{
				Msg: "invalid",
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus() != tt.wantCode {
				t.Errorf("HTTPStatus() = %d; want %d", tt.err.HTTPStatus(), tt.wantCode)
			}

			resp := tt.err.ToErrorResponse()
			if resp.Message == "" {
				t.Error("ToErrorResponse().Message should not be empty")
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrAuthUserNotFound",
			err:  happydns.ErrAuthUserNotFound,
			msg:  "user not found",
		},
		{
			name: "ErrDomainNotFound",
			err:  happydns.ErrDomainNotFound,
			msg:  "domain not found",
		},
		{
			name: "ErrDomainLogNotFound",
			err:  happydns.ErrDomainLogNotFound,
			msg:  "domain log not found",
		},
		{
			name: "ErrProviderNotFound",
			err:  happydns.ErrProviderNotFound,
			msg:  "provider not found",
		},
		{
			name: "ErrSessionNotFound",
			err:  happydns.ErrSessionNotFound,
			msg:  "session not found",
		},
		{
			name: "ErrUserNotFound",
			err:  happydns.ErrUserNotFound,
			msg:  "user not found",
		},
		{
			name: "ErrUserAlreadyExist",
			err:  happydns.ErrUserAlreadyExist,
			msg:  "user already exists",
		},
		{
			name: "ErrZoneNotFound",
			err:  happydns.ErrZoneNotFound,
			msg:  "zone not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("Error message = %q; want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}
