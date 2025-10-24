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

package session_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

func createTestUser(t *testing.T, store *inmemory.InMemoryStorage, email string) *happydns.User {
	user := &happydns.User{
		Id:    happydns.Identifier([]byte("user-" + email)),
		Email: email,
	}
	if err := store.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func Test_CreateUserSession(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	sess, err := sessionService.CreateUserSession(user, "Test session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sess.Id == "" {
		t.Error("expected session ID to be set")
	}
	if !sess.IdUser.Equals(user.Id) {
		t.Errorf("expected session IdUser to be %v, got %v", user.Id, sess.IdUser)
	}
	if sess.Description != "Test session" {
		t.Errorf("expected description 'Test session', got %s", sess.Description)
	}
	if sess.IssuedAt.IsZero() {
		t.Error("expected IssuedAt to be set")
	}
	if sess.ExpiresOn.IsZero() {
		t.Error("expected ExpiresOn to be set")
	}

	// Verify session is stored in database
	stored, err := mem.GetSession(sess.Id)
	if err != nil {
		t.Fatalf("expected stored session, got error: %v", err)
	}
	if stored.Description != "Test session" {
		t.Errorf("expected stored description to be 'Test session', got %s", stored.Description)
	}
}

func Test_GetUserSession(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create a session
	createdSession, err := sessionService.CreateUserSession(user, "Test session")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Retrieve the session
	retrievedSession, err := sessionService.GetUserSession(user, createdSession.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrievedSession.Id != createdSession.Id {
		t.Errorf("expected session ID %s, got %s", createdSession.Id, retrievedSession.Id)
	}
	if retrievedSession.Description != "Test session" {
		t.Errorf("expected description 'Test session', got %s", retrievedSession.Description)
	}
}

func Test_GetUserSession_WrongUser(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create a session for user1
	createdSession, err := sessionService.CreateUserSession(user1, "User1 session")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Try to retrieve the session as user2
	_, err = sessionService.GetUserSession(user2, createdSession.Id)
	if err == nil {
		t.Error("expected error when retrieving another user's session")
	}
	if err != happydns.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func Test_GetUserSession_NotFound(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	_, err := sessionService.GetUserSession(user, "nonexistent-session-id")
	if err == nil {
		t.Error("expected error when retrieving nonexistent session")
	}
	if err != happydns.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func Test_ListUserSessions(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create multiple sessions
	_, err := sessionService.CreateUserSession(user, "Session 1")
	if err != nil {
		t.Fatalf("unexpected error creating session 1: %v", err)
	}
	_, err = sessionService.CreateUserSession(user, "Session 2")
	if err != nil {
		t.Fatalf("unexpected error creating session 2: %v", err)
	}
	_, err = sessionService.CreateUserSession(user, "Session 3")
	if err != nil {
		t.Fatalf("unexpected error creating session 3: %v", err)
	}

	// List sessions
	sessions, err := sessionService.ListUserSessions(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func Test_ListUserSessions_MultipleUsers(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create sessions for user1
	_, err := sessionService.CreateUserSession(user1, "User1 Session 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = sessionService.CreateUserSession(user1, "User1 Session 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create session for user2
	_, err = sessionService.CreateUserSession(user2, "User2 Session 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List sessions for user1
	user1Sessions, err := sessionService.ListUserSessions(user1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user1Sessions) != 2 {
		t.Errorf("expected 2 sessions for user1, got %d", len(user1Sessions))
	}

	// List sessions for user2
	user2Sessions, err := sessionService.ListUserSessions(user2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user2Sessions) != 1 {
		t.Errorf("expected 1 session for user2, got %d", len(user2Sessions))
	}
}

func Test_UpdateUserSession(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create a session
	createdSession, err := sessionService.CreateUserSession(user, "Original description")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Update the session
	err = sessionService.UpdateUserSession(user, createdSession.Id, func(sess *happydns.Session) {
		sess.Description = "Updated description"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the session was updated
	updated, err := sessionService.GetUserSession(user, createdSession.Id)
	if err != nil {
		t.Fatalf("unexpected error retrieving updated session: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %s", updated.Description)
	}
	if updated.ModifiedOn.IsZero() {
		t.Error("expected ModifiedOn to be set after update")
	}
}

func Test_UpdateUserSession_PreventIdChange(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create a session
	createdSession, err := sessionService.CreateUserSession(user, "Test session")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Try to change the session ID
	err = sessionService.UpdateUserSession(user, createdSession.Id, func(sess *happydns.Session) {
		sess.Id = "new-session-id"
	})
	if err == nil {
		t.Error("expected error when trying to change session ID")
	}
	if err.Error() != "you cannot change the session identifier" {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func Test_UpdateUserSession_WrongUser(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create a session for user1
	createdSession, err := sessionService.CreateUserSession(user1, "User1 session")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Try to update the session as user2
	err = sessionService.UpdateUserSession(user2, createdSession.Id, func(sess *happydns.Session) {
		sess.Description = "Hijacked"
	})
	if err == nil {
		t.Error("expected error when updating another user's session")
	}
}

func Test_DeleteUserSession(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create a session
	createdSession, err := sessionService.CreateUserSession(user, "Test session")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Delete the session
	err = sessionService.DeleteUserSession(user, createdSession.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the session was deleted
	_, err = sessionService.GetUserSession(user, createdSession.Id)
	if err == nil {
		t.Error("expected error when retrieving deleted session")
	}
	if err != happydns.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func Test_DeleteUserSession_WrongUser(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create a session for user1
	createdSession, err := sessionService.CreateUserSession(user1, "User1 session")
	if err != nil {
		t.Fatalf("unexpected error creating session: %v", err)
	}

	// Try to delete the session as user2
	err = sessionService.DeleteUserSession(user2, createdSession.Id)
	if err == nil {
		t.Error("expected error when deleting another user's session")
	}

	// Verify the session still exists
	_, err = sessionService.GetUserSession(user1, createdSession.Id)
	if err != nil {
		t.Errorf("session should still exist, got error: %v", err)
	}
}

func Test_CloseUserSessions(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create multiple sessions
	_, err := sessionService.CreateUserSession(user, "Session 1")
	if err != nil {
		t.Fatalf("unexpected error creating session 1: %v", err)
	}
	_, err = sessionService.CreateUserSession(user, "Session 2")
	if err != nil {
		t.Fatalf("unexpected error creating session 2: %v", err)
	}
	_, err = sessionService.CreateUserSession(user, "Session 3")
	if err != nil {
		t.Fatalf("unexpected error creating session 3: %v", err)
	}

	// Close all sessions
	err = sessionService.CloseUserSessions(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all sessions were deleted
	sessions, err := sessionService.ListUserSessions(user)
	if err != nil {
		t.Fatalf("unexpected error listing sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after closing all, got %d", len(sessions))
	}
}

func Test_CloseUserSessions_MultipleUsers(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create sessions for both users
	_, err := sessionService.CreateUserSession(user1, "User1 Session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	user2Session, err := sessionService.CreateUserSession(user2, "User2 Session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close all sessions for user1
	err = sessionService.CloseUserSessions(user1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify user1 sessions are deleted
	user1Sessions, err := sessionService.ListUserSessions(user1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user1Sessions) != 0 {
		t.Errorf("expected 0 sessions for user1, got %d", len(user1Sessions))
	}

	// Verify user2 session still exists
	user2Sessions, err := sessionService.ListUserSessions(user2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user2Sessions) != 1 {
		t.Errorf("expected 1 session for user2, got %d", len(user2Sessions))
	}
	if len(user2Sessions) > 0 && user2Sessions[0].Id != user2Session.Id {
		t.Error("user2's session was incorrectly deleted")
	}
}

type testUserInfo struct {
	id happydns.Identifier
}

func (u testUserInfo) GetUserId() happydns.Identifier { return u.id }
func (u testUserInfo) GetEmail() string               { return "" }
func (u testUserInfo) JoinNewsletter() bool           { return false }

func Test_CloseAll_UserInfoInterface(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	userID := happydns.Identifier([]byte("user-123"))
	user := &happydns.User{
		Id:    userID,
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create sessions
	_, err := sessionService.CreateUserSession(user, "Session 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = sessionService.CreateUserSession(user, "Session 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close all sessions using UserInfo interface
	userInfo := testUserInfo{id: userID}
	err = sessionService.CloseAll(userInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all sessions were deleted
	sessions, err := sessionService.ListUserSessions(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func Test_ByID(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	userID := happydns.Identifier([]byte("user-123"))
	user := &happydns.User{
		Id:    userID,
		Email: "test@example.com",
	}
	if err := mem.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create sessions
	_, err := sessionService.CreateUserSession(user, "Session 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close sessions by user ID
	err = sessionService.ByID(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all sessions were deleted
	sessions, err := sessionService.ListUserSessions(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func Test_NewSessionID(t *testing.T) {
	// Generate multiple session IDs to ensure they're unique
	id1 := session.NewSessionID()
	id2 := session.NewSessionID()
	id3 := session.NewSessionID()

	if id1 == "" || id2 == "" || id3 == "" {
		t.Error("expected non-empty session IDs")
	}

	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Error("expected unique session IDs")
	}

	// Session IDs should be base32 encoded (no padding)
	if len(id1) == 0 {
		t.Error("expected session ID to have length")
	}
}

func Test_SessionExpiration(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	sessionService := session.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// Create a session
	sess, err := sessionService.CreateUserSession(user, "Test session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify expiration is set to approximately 1 year from now
	expectedExpiration := time.Now().Add(24 * 365 * time.Hour)
	timeDiff := sess.ExpiresOn.Sub(expectedExpiration)
	if timeDiff < -1*time.Minute || timeDiff > 1*time.Minute {
		t.Errorf("expected expiration to be around 1 year from now, got %v", sess.ExpiresOn)
	}
}
