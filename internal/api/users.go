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

package api

import (
	"context"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// ClearSession removes the content of the current user's session.
func (s *Server) ClearSession(ctx context.Context, request ClearSessionRequestObject) (ClearSessionResponseObject, error) {
	// TODO: Implement session clearing
	return ClearSession401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// GetSession gets the content of the current user's session.
func (s *Server) GetSession(ctx context.Context, request GetSessionRequestObject) (GetSessionResponseObject, error) {
	// TODO: Get user from context and session ID
	// user := &happydns.User{} // Placeholder
	// session, err := s.dependancies.SessionUsecase().GetUserSession(user, sessionID)
	return GetSession401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// DeleteAllSessions closes all sessions for a given user.
func (s *Server) DeleteAllSessions(ctx context.Context, request DeleteAllSessionsRequestObject) (DeleteAllSessionsResponseObject, error) {
	// TODO: Implement delete all sessions
	return DeleteAllSessions401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// ListSessions lists the sessions open for the current user.
func (s *Server) ListSessions(ctx context.Context, request ListSessionsRequestObject) (ListSessionsResponseObject, error) {
	// TODO: Implement list sessions
	return ListSessions401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// CreateSession creates a new session for the current user.
func (s *Server) CreateSession(ctx context.Context, request CreateSessionRequestObject) (CreateSessionResponseObject, error) {
	// TODO: Implement create session
	return CreateSession401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// DeleteSession deletes a session owned by the current user.
func (s *Server) DeleteSession(ctx context.Context, request DeleteSessionRequestObject) (DeleteSessionResponseObject, error) {
	// TODO: Implement delete session
	return DeleteSession401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// UpdateSession updates a session owned by the current user.
func (s *Server) UpdateSession(ctx context.Context, request UpdateSessionRequestObject) (UpdateSessionResponseObject, error) {
	// TODO: Implement update session
	return UpdateSession401JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// UserSpecialAction handles account recovery or email validation link sending.
func (s *Server) UserSpecialAction(ctx context.Context, request UserSpecialActionRequestObject) (UserSpecialActionResponseObject, error) {
	// TODO: Implement user special action (recovery, email validation)
	return UserSpecialAction200JSONResponse(happydns.ErrorResponse{
		Message: "Perhaps something happen",
	}), nil
}

// RegisterUser registers a new happyDomain account.
func (s *Server) RegisterUser(ctx context.Context, request RegisterUserRequestObject) (RegisterUserResponseObject, error) {
	var user happydns.User
	user.Email = request.Body.Email
	// TODO: Map other fields from UserRegistration to User

	createdUser, err := s.dependancies.UserUsecase().CreateUser(&user)
	if err != nil {
		return RegisterUser500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to register user: %s", err.Error()),
		}), nil
	}

	return RegisterUser200JSONResponse(*createdUser), nil
}

// GetUser shows a user from the database.
func (s *Server) GetUser(ctx context.Context, request GetUserRequestObject) (GetUserResponseObject, error) {
	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return GetUser500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	user, err := s.dependancies.UserUsecase().GetUser(userId)
	if err != nil {
		return GetUser500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("User not found: %s", err.Error()),
		}), nil
	}

	return GetUser200JSONResponse(*user), nil
}

// GetUserAvatar returns a unique avatar for the user.
func (s *Server) GetUserAvatar(ctx context.Context, request GetUserAvatarRequestObject) (GetUserAvatarResponseObject, error) {
	// TODO: Implement avatar generation
	// userId, err := happydns.NewIdentifierFromString(request.UserId)
	// size := 64
	// if request.Params.Size != nil {
	// 	size = *request.Params.Size
	// }
	return GetUserAvatar500JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// DeleteUser deletes the account related to the given user.
func (s *Server) DeleteUser(ctx context.Context, request DeleteUserRequestObject) (DeleteUserResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return DeleteUser401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return DeleteUser400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	// Verify user is deleting their own account
	if !user.Id.Equals(userId) {
		return DeleteUser403JSONResponse(happydns.ErrorResponse{
			Message: "You can only delete your own account",
		}), nil
	}

	// TODO: Verify password from request.Body.Password
	// err = s.dependancies.UserUsecase().DeleteUser(userId)
	// if err != nil {
	// 	return DeleteUser500JSONResponse{ErrorResponse: happydns.ErrorResponse{
	// 		Message: fmt.Sprintf("Failed to delete user: %s", err.Error()),
	// 	}}, nil
	// }

	_ = userId // Suppress unused variable warning for now

	return DeleteUser204Response{}, nil
}

// ValidateUserEmail validates the email address of the user.
func (s *Server) ValidateUserEmail(ctx context.Context, request ValidateUserEmailRequestObject) (ValidateUserEmailResponseObject, error) {
	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return ValidateUserEmail400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	// TODO: Validate email using request.Body.ValidationKey
	_ = userId // Suppress unused variable warning for now

	return ValidateUserEmail204Response{}, nil
}

// ChangeUserPassword changes the password of the given account.
func (s *Server) ChangeUserPassword(ctx context.Context, request ChangeUserPasswordRequestObject) (ChangeUserPasswordResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return ChangeUserPassword401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return ChangeUserPassword400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	// Verify user is changing their own password
	if !user.Id.Equals(userId) {
		return ChangeUserPassword403JSONResponse(happydns.ErrorResponse{
			Message: "You can only change your own password",
		}), nil
	}

	// TODO: Verify current password and set new password
	// err = s.dependancies.UserUsecase().ChangePassword(userId, request.Body.OldPassword, request.Body.NewPassword)
	_ = userId // Suppress unused variable warning for now

	return ChangeUserPassword204Response{}, nil
}

// RecoverUserAccount performs account recovery by resetting the password.
func (s *Server) RecoverUserAccount(ctx context.Context, request RecoverUserAccountRequestObject) (RecoverUserAccountResponseObject, error) {
	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return RecoverUserAccount400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	// TODO: Implement account recovery using request.Body.RecoveryKey and request.Body.NewPassword
	_ = userId // Suppress unused variable warning for now

	return RecoverUserAccount204Response{}, nil
}

// GetUserSettings retrieves the user's settings.
func (s *Server) GetUserSettings(ctx context.Context, request GetUserSettingsRequestObject) (GetUserSettingsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetUserSettings401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return GetUserSettings403JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	// Verify user is accessing their own settings
	if !user.Id.Equals(userId) {
		return GetUserSettings403JSONResponse(happydns.ErrorResponse{
			Message: "You can only access your own settings",
		}), nil
	}

	// TODO: Get user settings
	_ = userId // Suppress unused variable warning for now
	settings := happydns.UserSettings{}

	return GetUserSettings200JSONResponse(settings), nil
}

// UpdateUserSettings updates the user's settings.
func (s *Server) UpdateUserSettings(ctx context.Context, request UpdateUserSettingsRequestObject) (UpdateUserSettingsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return UpdateUserSettings401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	userId, err := happydns.NewIdentifierFromString(request.UserId)
	if err != nil {
		return UpdateUserSettings403JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid user ID: %s", err.Error()),
		}), nil
	}

	// Verify user is updating their own settings
	if user.Id.Equals(userId) {
		return UpdateUserSettings403JSONResponse(happydns.ErrorResponse{
			Message: "You can only update your own settings",
		}), nil
	}

	// TODO: Update user settings with request.Body
	_ = userId // Suppress unused variable warning for now

	return UpdateUserSettings200JSONResponse(*request.Body), nil
}
