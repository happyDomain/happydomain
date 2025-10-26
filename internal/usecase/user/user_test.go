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

package user_test

import (
	"bytes"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/model"
)

// Mock implementations for testing
type mockNewsletterSubscriptor struct {
	subscribed []happydns.UserInfo
	shouldFail bool
}

func (m *mockNewsletterSubscriptor) SubscribeToNewsletter(uinfo happydns.UserInfo) error {
	if m.shouldFail {
		return happydns.InternalError{Err: nil, UserMessage: "newsletter subscription failed"}
	}
	m.subscribed = append(m.subscribed, uinfo)
	return nil
}

type mockSessionCloser struct {
	closedUserIDs []happydns.Identifier
}

func (m *mockSessionCloser) ByID(userid happydns.Identifier) error {
	m.closedUserIDs = append(m.closedUserIDs, userid)
	return nil
}

func (m *mockSessionCloser) CloseAll(uinfo happydns.UserInfo) error {
	m.closedUserIDs = append(m.closedUserIDs, uinfo.GetUserId())
	return nil
}

type testUserInfo struct {
	id             happydns.Identifier
	email          string
	joinNewsletter bool
}

func (u testUserInfo) GetUserId() happydns.Identifier { return u.id }
func (u testUserInfo) GetEmail() string               { return u.email }
func (u testUserInfo) JoinNewsletter() bool           { return u.joinNewsletter }

func createTestService(t *testing.T) (*user.Service, *inmemory.InMemoryStorage, *mockNewsletterSubscriptor, *mockSessionCloser) {
	mem, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}

	cfg := &happydns.Options{
		DisableRegistration: false,
	}
	sessionService := sessionUC.NewService(mem)
	authUserService := authuserUC.NewAuthUserUsecases(cfg, nil, mem, sessionService)

	newsletter := &mockNewsletterSubscriptor{}
	sessionCloser := &mockSessionCloser{}

	service := user.NewUserUsecases(mem, newsletter, authUserService, sessionCloser)

	return service, mem, newsletter, sessionCloser
}

func Test_CreateUser(t *testing.T) {
	service, _, newsletter, _ := createTestService(t)

	userInfo := testUserInfo{
		id:             happydns.Identifier([]byte("user-123")),
		email:          "test@example.com",
		joinNewsletter: false,
	}

	user, err := service.CreateUser(userInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !user.Id.Equals(userInfo.id) {
		t.Errorf("expected user ID to be %v, got %v", userInfo.id, user.Id)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", user.Email)
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if user.LastSeen.IsZero() {
		t.Error("expected LastSeen to be set")
	}
	if len(newsletter.subscribed) != 0 {
		t.Error("expected user not to be subscribed to newsletter")
	}
}

func Test_CreateUser_WithNewsletter(t *testing.T) {
	service, _, newsletter, _ := createTestService(t)

	userInfo := testUserInfo{
		id:             happydns.Identifier([]byte("user-123")),
		email:          "test@example.com",
		joinNewsletter: true,
	}

	user, err := service.CreateUser(userInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", user.Email)
	}
	if len(newsletter.subscribed) != 1 {
		t.Errorf("expected 1 newsletter subscription, got %d", len(newsletter.subscribed))
	}
	if len(newsletter.subscribed) > 0 && newsletter.subscribed[0].GetEmail() != "test@example.com" {
		t.Errorf("expected newsletter subscription for test@example.com, got %s", newsletter.subscribed[0].GetEmail())
	}
}

func Test_CreateUser_NoEmail(t *testing.T) {
	service, _, _, _ := createTestService(t)

	userInfo := testUserInfo{
		id:             happydns.Identifier([]byte("user-123")),
		email:          "",
		joinNewsletter: false,
	}

	_, err := service.CreateUser(userInfo)
	if err == nil {
		t.Error("expected error when creating user without email")
	}
	if err.Error() != "user email is required" {
		t.Errorf("expected 'user email is required' error, got: %v", err)
	}
}

func Test_CreateUser_NewsletterFailure(t *testing.T) {
	service, _, newsletter, _ := createTestService(t)
	newsletter.shouldFail = true

	userInfo := testUserInfo{
		id:             happydns.Identifier([]byte("user-123")),
		email:          "test@example.com",
		joinNewsletter: true,
	}

	user, err := service.CreateUser(userInfo)
	if err == nil {
		t.Error("expected error when newsletter subscription fails")
	}
	// User should still be created even if newsletter fails
	if user == nil {
		t.Error("expected user to be returned even when newsletter fails")
	}
}

func Test_GetUser(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create a user directly in storage
	userID := happydns.Identifier([]byte("user-123"))
	createdUser := &happydns.User{
		Id:    userID,
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(createdUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Retrieve the user
	retrievedUser, err := service.GetUser(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !retrievedUser.Id.Equals(userID) {
		t.Errorf("expected user ID %v, got %v", userID, retrievedUser.Id)
	}
	if retrievedUser.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", retrievedUser.Email)
	}
}

func Test_GetUser_NotFound(t *testing.T) {
	service, _, _, _ := createTestService(t)

	_, err := service.GetUser(happydns.Identifier([]byte("nonexistent")))
	if err == nil {
		t.Error("expected error when retrieving nonexistent user")
	}
}

func Test_GetUserByEmail(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create a user directly in storage
	createdUser := &happydns.User{
		Id:    happydns.Identifier([]byte("user-123")),
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(createdUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Retrieve the user by email
	retrievedUser, err := service.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrievedUser.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", retrievedUser.Email)
	}
}

func Test_GetUserByEmail_NotFound(t *testing.T) {
	service, _, _, _ := createTestService(t)

	_, err := service.GetUserByEmail("nonexistent@example.com")
	if err == nil {
		t.Error("expected error when retrieving user with nonexistent email")
	}
}

func Test_UpdateUser(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create a user directly in storage
	userID := happydns.Identifier([]byte("user-123"))
	createdUser := &happydns.User{
		Id:    userID,
		Email: "original@example.com",
	}
	if err := mem.CreateOrUpdateUser(createdUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Update the user
	err := service.UpdateUser(userID, func(u *happydns.User) {
		u.Email = "updated@example.com"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the user was updated
	updatedUser, err := service.GetUser(userID)
	if err != nil {
		t.Fatalf("unexpected error retrieving updated user: %v", err)
	}
	if updatedUser.Email != "updated@example.com" {
		t.Errorf("expected email 'updated@example.com', got %s", updatedUser.Email)
	}
}

func Test_UpdateUser_PreventIdChange(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create a user directly in storage
	userID := happydns.Identifier([]byte("user-123"))
	createdUser := &happydns.User{
		Id:    userID,
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(createdUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Try to change the user ID
	err := service.UpdateUser(userID, func(u *happydns.User) {
		u.Id = happydns.Identifier([]byte("new-id"))
	})
	if err == nil {
		t.Error("expected error when trying to change user ID")
	}

	// Check for ValidationError
	if _, ok := err.(happydns.ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func Test_UpdateUser_NotFound(t *testing.T) {
	service, _, _, _ := createTestService(t)

	err := service.UpdateUser(happydns.Identifier([]byte("nonexistent")), func(u *happydns.User) {
		u.Email = "updated@example.com"
	})
	if err == nil {
		t.Error("expected error when updating nonexistent user")
	}
}

func Test_ChangeUserSettings(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create a user with default settings
	user := &happydns.User{
		Id:       happydns.Identifier([]byte("user-123")),
		Email:    "test@example.com",
		Settings: *happydns.DefaultUserSettings(),
	}
	if err := mem.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Change settings
	newSettings := happydns.UserSettings{
		Language: "fr",
	}
	err := service.ChangeUserSettings(user, newSettings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify settings were changed
	if user.Settings.Language != "fr" {
		t.Errorf("expected language 'fr', got %s", user.Settings.Language)
	}

	// Verify in storage
	storedUser, err := service.GetUser(user.Id)
	if err != nil {
		t.Fatalf("unexpected error retrieving user: %v", err)
	}
	if storedUser.Settings.Language != "fr" {
		t.Errorf("expected stored language 'fr', got %s", storedUser.Settings.Language)
	}
}

func Test_DeleteUser(t *testing.T) {
	service, mem, _, sessionCloser := createTestService(t)

	// Create a user (external account, no auth user)
	userID := happydns.Identifier([]byte("user-123"))
	user := &happydns.User{
		Id:    userID,
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Delete the user
	err := service.DeleteUser(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify user was deleted
	_, err = service.GetUser(userID)
	if err == nil {
		t.Error("expected error when retrieving deleted user")
	}

	// Verify sessions were closed
	if len(sessionCloser.closedUserIDs) != 1 {
		t.Errorf("expected 1 session close call, got %d", len(sessionCloser.closedUserIDs))
	}
	if len(sessionCloser.closedUserIDs) > 0 && !sessionCloser.closedUserIDs[0].Equals(userID) {
		t.Error("expected sessions to be closed for the deleted user")
	}
}

func Test_DeleteUser_WithAuthUser(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create an auth user (local account)
	authUser := &happydns.UserAuth{
		Email:    "test@example.com",
		Password: []byte("hashed-password"),
	}
	if err := mem.CreateAuthUser(authUser); err != nil {
		t.Fatalf("failed to create auth user: %v", err)
	}

	// Try to delete the user (should fail for local accounts)
	err := service.DeleteUser(authUser.Id)
	if err == nil {
		t.Error("expected error when deleting user with local auth")
	}
	if err.Error() != "This route is for external account only. Please use the route ./delete instead." {
		t.Errorf("unexpected error message: %v", err)
	}
}

func Test_GenerateUserAvatar(t *testing.T) {
	service, mem, _, _ := createTestService(t)

	// Create a user
	user := &happydns.User{
		Id:    happydns.Identifier([]byte("user-123")),
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Generate avatar
	var buf bytes.Buffer
	err := service.GenerateUserAvatar(user, 64, &buf)
	if err != nil {
		t.Fatalf("unexpected error generating avatar: %v", err)
	}

	// Verify some data was written
	if buf.Len() == 0 {
		t.Error("expected avatar data to be written, got empty buffer")
	}
}
