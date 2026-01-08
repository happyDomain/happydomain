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

func TestSessionClearSession(t *testing.T) {
	userId := happydns.Identifier{0x01, 0x02, 0x03}

	session := &happydns.Session{
		Id:          "session123",
		IdUser:      userId,
		Description: "test session",
		IssuedAt:    time.Now(),
		ExpiresOn:   time.Now().Add(24 * time.Hour),
		ModifiedOn:  time.Now(),
		Content:     "sensitive data",
	}

	if session.Content == "" {
		t.Fatal("Test setup failed: session content should not be empty")
	}

	session.ClearSession()

	if session.Content != "" {
		t.Errorf("ClearSession() Content = %q; want empty string", session.Content)
	}

	if session.Id != "session123" {
		t.Error("ClearSession() should not modify session ID")
	}

	if !session.IdUser.Equals(userId) {
		t.Error("ClearSession() should not modify user ID")
	}

	if session.Description != "test session" {
		t.Error("ClearSession() should not modify description")
	}
}

func TestSessionClearSessionAlreadyEmpty(t *testing.T) {
	session := &happydns.Session{
		Id:      "session123",
		IdUser:  happydns.Identifier{0x01},
		Content: "",
	}

	session.ClearSession()

	if session.Content != "" {
		t.Errorf("ClearSession() on empty content: Content = %q; want empty string", session.Content)
	}
}

func TestSessionClearSessionMultipleTimes(t *testing.T) {
	session := &happydns.Session{
		Id:      "session123",
		IdUser:  happydns.Identifier{0x01},
		Content: "data",
	}

	session.ClearSession()
	if session.Content != "" {
		t.Error("First ClearSession() failed")
	}

	session.ClearSession()
	if session.Content != "" {
		t.Error("Second ClearSession() on already cleared session failed")
	}
}

func TestSessionStructFields(t *testing.T) {
	sessionId := "test-session-123"
	userId := happydns.Identifier{0xaa, 0xbb, 0xcc}
	description := "Test Session Description"
	issuedAt := time.Now()
	expiresOn := time.Now().Add(48 * time.Hour)
	modifiedOn := time.Now().Add(1 * time.Hour)
	content := "session content data"

	session := &happydns.Session{
		Id:          sessionId,
		IdUser:      userId,
		Description: description,
		IssuedAt:    issuedAt,
		ExpiresOn:   expiresOn,
		ModifiedOn:  modifiedOn,
		Content:     content,
	}

	if session.Id != sessionId {
		t.Errorf("Session.Id = %q; want %q", session.Id, sessionId)
	}

	if !session.IdUser.Equals(userId) {
		t.Errorf("Session.IdUser = %v; want %v", session.IdUser, userId)
	}

	if session.Description != description {
		t.Errorf("Session.Description = %q; want %q", session.Description, description)
	}

	if !session.IssuedAt.Equal(issuedAt) {
		t.Errorf("Session.IssuedAt = %v; want %v", session.IssuedAt, issuedAt)
	}

	if !session.ExpiresOn.Equal(expiresOn) {
		t.Errorf("Session.ExpiresOn = %v; want %v", session.ExpiresOn, expiresOn)
	}

	if !session.ModifiedOn.Equal(modifiedOn) {
		t.Errorf("Session.ModifiedOn = %v; want %v", session.ModifiedOn, modifiedOn)
	}

	if session.Content != content {
		t.Errorf("Session.Content = %q; want %q", session.Content, content)
	}
}

func TestSessionZeroValues(t *testing.T) {
	session := &happydns.Session{}

	if session.Id != "" {
		t.Errorf("Session zero value should have empty Id, got %q", session.Id)
	}

	if !session.IdUser.IsEmpty() {
		t.Error("Session zero value should have empty IdUser")
	}

	if session.Description != "" {
		t.Errorf("Session zero value should have empty Description, got %q", session.Description)
	}

	if !session.IssuedAt.IsZero() {
		t.Error("Session zero value should have zero IssuedAt")
	}

	if !session.ExpiresOn.IsZero() {
		t.Error("Session zero value should have zero ExpiresOn")
	}

	if !session.ModifiedOn.IsZero() {
		t.Error("Session zero value should have zero ModifiedOn")
	}

	if session.Content != "" {
		t.Errorf("Session zero value should have empty Content, got %q", session.Content)
	}
}

func TestSessionClearSessionPreservesTimestamps(t *testing.T) {
	issuedAt := time.Now().Add(-2 * time.Hour)
	expiresOn := time.Now().Add(22 * time.Hour)
	modifiedOn := time.Now().Add(-1 * time.Hour)

	session := &happydns.Session{
		Id:          "session123",
		IdUser:      happydns.Identifier{0x01},
		Description: "test",
		IssuedAt:    issuedAt,
		ExpiresOn:   expiresOn,
		ModifiedOn:  modifiedOn,
		Content:     "data to clear",
	}

	session.ClearSession()

	if !session.IssuedAt.Equal(issuedAt) {
		t.Error("ClearSession() should not modify IssuedAt")
	}

	if !session.ExpiresOn.Equal(expiresOn) {
		t.Error("ClearSession() should not modify ExpiresOn")
	}

	if !session.ModifiedOn.Equal(modifiedOn) {
		t.Error("ClearSession() should not modify ModifiedOn")
	}
}

func TestSessionClearSessionWithLargeContent(t *testing.T) {
	largeContent := string(make([]byte, 10000))

	session := &happydns.Session{
		Id:      "session123",
		IdUser:  happydns.Identifier{0x01},
		Content: largeContent,
	}

	session.ClearSession()

	if session.Content != "" {
		t.Error("ClearSession() should clear even large content")
	}
}
