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
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

func TestUserGetUserId(t *testing.T) {
	userId := happydns.Identifier{0x01, 0x02, 0x03, 0x04}

	user := &happydns.User{
		Id:    userId,
		Email: "test@example.com",
	}

	result := user.GetUserId()

	if !result.Equals(userId) {
		t.Errorf("GetUserId() = %v; want %v", result, userId)
	}
}

func TestUserGetUserIdEmpty(t *testing.T) {
	user := &happydns.User{
		Id:    happydns.Identifier{},
		Email: "test@example.com",
	}

	result := user.GetUserId()

	if !result.IsEmpty() {
		t.Errorf("GetUserId() should return empty identifier, got %v", result)
	}
}

func TestUserGetEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "standard email",
			email: "test@example.com",
		},
		{
			name:  "email with subdomain",
			email: "user@mail.example.com",
		},
		{
			name:  "email with plus",
			email: "user+tag@example.com",
		},
		{
			name:  "empty email",
			email: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &happydns.User{
				Id:    happydns.Identifier{0x01},
				Email: tt.email,
			}

			result := user.GetEmail()

			if result != tt.email {
				t.Errorf("GetEmail() = %q; want %q", result, tt.email)
			}
		})
	}
}

func TestUserJoinNewsletter(t *testing.T) {
	user := &happydns.User{
		Id:    happydns.Identifier{0x01, 0x02},
		Email: "test@example.com",
	}

	result := user.JoinNewsletter()

	if result != false {
		t.Errorf("JoinNewsletter() = %v; want false", result)
	}
}

func TestUserJoinNewsletterAlwaysFalse(t *testing.T) {
	tests := []struct {
		name string
		user *happydns.User
	}{
		{
			name: "user with id",
			user: &happydns.User{
				Id:    happydns.Identifier{0x01, 0x02, 0x03},
				Email: "test1@example.com",
			},
		},
		{
			name: "user without id",
			user: &happydns.User{
				Id:    happydns.Identifier{},
				Email: "test2@example.com",
			},
		},
		{
			name: "user with settings",
			user: &happydns.User{
				Id:       happydns.Identifier{0x04, 0x05},
				Email:    "test3@example.com",
				Settings: happydns.UserSettings{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.JoinNewsletter()
			if result != false {
				t.Errorf("JoinNewsletter() = %v; want false", result)
			}
		})
	}
}

func TestUserStructFields(t *testing.T) {
	userId := happydns.Identifier{0x11, 0x22, 0x33}
	email := "test@example.com"
	createdAt := time.Now()
	lastSeen := time.Now().Add(24 * time.Hour)

	user := &happydns.User{
		Id:        userId,
		Email:     email,
		CreatedAt: createdAt,
		LastSeen:  lastSeen,
		Settings:  happydns.UserSettings{},
	}

	if !user.Id.Equals(userId) {
		t.Errorf("User.Id = %v; want %v", user.Id, userId)
	}

	if user.Email != email {
		t.Errorf("User.Email = %q; want %q", user.Email, email)
	}

	if !user.CreatedAt.Equal(createdAt) {
		t.Errorf("User.CreatedAt = %v; want %v", user.CreatedAt, createdAt)
	}

	if !user.LastSeen.Equal(lastSeen) {
		t.Errorf("User.LastSeen = %v; want %v", user.LastSeen, lastSeen)
	}
}

func TestUserZeroValues(t *testing.T) {
	user := &happydns.User{}

	if !user.Id.IsEmpty() {
		t.Error("User zero value should have empty Id")
	}

	if user.Email != "" {
		t.Errorf("User zero value should have empty Email, got %q", user.Email)
	}

	if !user.CreatedAt.IsZero() {
		t.Error("User zero value should have zero CreatedAt")
	}

	if !user.LastSeen.IsZero() {
		t.Error("User zero value should have zero LastSeen")
	}
}
