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

package authuser_test

import (
	"fmt"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

// MockCloseUserSessionsUsecase is a mock implementation of CloseUserSessionsUsecase.
type MockCloseUserSessionsUsecase struct {
	CloseAllFunc func(user happydns.UserInfo) error
}

func (m *MockCloseUserSessionsUsecase) CloseAll(user happydns.UserInfo) error {
	return m.CloseAllFunc(user)
}

func (m *MockCloseUserSessionsUsecase) ByID(userID happydns.Identifier) error {
	return m.CloseAll(&happydns.UserAuth{Id: userID})
}

func TestDeleteAuthUserUsecase_Delete(t *testing.T) {
	// Create an in-memory storage
	store, _ := inmemory.NewInMemoryStorage()

	// Create a mock for CloseUserSessionsUsecase
	mockCloseUserSessions := &MockCloseUserSessionsUsecase{
		CloseAllFunc: func(user happydns.UserInfo) error {
			return nil
		},
	}

	// Create an instance of DeleteAuthUserUsecase
	uc := authuser.NewDeleteAuthUserUsecase(store, mockCloseUserSessions)

	// Create a test user
	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("test-password")

	// Add the user to the storage
	err := store.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case 1: Invalid password
	err = uc.Delete(user, "wrong-password")
	if err == nil || err.Error() != "invalid current password" {
		t.Errorf("Expected error 'invalid current password', got %v", err)
	}

	// Test case 2: Error in closing sessions
	mockCloseUserSessions.CloseAllFunc = func(user happydns.UserInfo) error {
		return fmt.Errorf("error closing sessions")
	}
	err = uc.Delete(user, "test-password")
	if err == nil || err.Error() != "unable to delete user sessions: error closing sessions" {
		t.Errorf("Expected error 'unable to delete user sessions: error closing sessions', got %v", err)
	}

	// Test case 3: Bad password when deleting user
	err = uc.Delete(user, "bad-password")
	if err == nil || err.Error() != "invalid current password" {
		t.Errorf("Expected error 'invalid current password', got %v", err)
	}

	// Test case 4: Successful deletion
	mockCloseUserSessions.CloseAllFunc = func(user happydns.UserInfo) error {
		return nil
	}
	err = uc.Delete(user, "test-password")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
