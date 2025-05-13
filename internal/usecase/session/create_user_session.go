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

package session

import (
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/securecookie"

	"git.happydns.org/happyDomain/model"
)

const SESSION_MAX_DURATION = 24 * 365 * time.Hour

type CreateUserSessionUsecase struct {
	store SessionStorage
}

func NewCreateUserSessionUsecase(store SessionStorage) *CreateUserSessionUsecase {
	return &CreateUserSessionUsecase{store: store}
}

func (uc *CreateUserSessionUsecase) Create(user *happydns.User, description string) (*happydns.Session, error) {
	sessid := NewSessionId()

	newsession := &happydns.Session{
		Id:          sessid,
		IdUser:      user.Id,
		Description: description,
		IssuedAt:    time.Now(),
		ExpiresOn:   time.Now().Add(SESSION_MAX_DURATION),
	}

	if err := uc.store.UpdateSession(newsession); err != nil {
		return nil, fmt.Errorf("unable to create new user session: %w", err)
	}

	return newsession, nil
}

func NewSessionId() string {
	return strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(64)), "=")
}
