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

package user

import (
	"fmt"
	"io"
	"time"

	"git.happydns.org/happyDomain/internal/avatar"
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	store             UserStorage
	newsletter        happydns.NewsletterSubscriptor
	authUser          happydns.AuthUserUsecase
	closeUserSessions happydns.SessionCloserUsecase
	onUserChanged     func(happydns.Identifier)
}

func NewUserUsecases(
	store UserStorage,
	newsletter happydns.NewsletterSubscriptor,
	authUser happydns.AuthUserUsecase,
	closeUserSessions happydns.SessionCloserUsecase,
) *Service {
	return &Service{
		store:             store,
		newsletter:        newsletter,
		authUser:          authUser,
		closeUserSessions: closeUserSessions,
	}
}

// SetOnUserChanged installs a callback invoked after any successful user
// update (via UpdateUser). This is used to invalidate caches that depend on
// user state, such as the scheduler's UserGater.
func (s *Service) SetOnUserChanged(fn func(happydns.Identifier)) {
	s.onUserChanged = fn
}

// CreateUser creates a new user with the given information.
func (s *Service) CreateUser(uinfo happydns.UserInfo) (*happydns.User, error) {
	if uinfo.GetEmail() == "" {
		return nil, fmt.Errorf("user email is required")
	}

	user := &happydns.User{
		Id:        uinfo.GetUserId(),
		Email:     uinfo.GetEmail(),
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Settings:  *happydns.DefaultUserSettings(),
	}

	if err := s.store.CreateOrUpdateUser(user); err != nil {
		return user, err
	}

	if uinfo.JoinNewsletter() {
		if err := s.newsletter.SubscribeToNewsletter(uinfo); err != nil {
			return user, fmt.Errorf("newsletter subscription failed: %w", err)
		}
	}

	return user, nil
}

// GetUser retrieves a user by their identifier.
func (s *Service) GetUser(userid happydns.Identifier) (*happydns.User, error) {
	return s.store.GetUser(userid)
}

// GetUserByEmail retrieves a user by their email address.
func (s *Service) GetUserByEmail(email string) (*happydns.User, error) {
	return s.store.GetUserByEmail(email)
}

// UpdateUser updates a user using the provided update function.
func (s *Service) UpdateUser(id happydns.Identifier, updateFn func(*happydns.User)) (*happydns.User, error) {
	user, err := s.store.GetUser(id)
	if err != nil {
		return nil, err
	}

	updateFn(user)

	if !user.Id.Equals(id) {
		return nil, happydns.ValidationError{Msg: "you cannot change the user identifier"}
	}

	if err := s.store.CreateOrUpdateUser(user); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("failed to update user: %w", err),
			UserMessage: "Sorry, we are currently unable to update your user. Please retry later.",
		}
	}

	if s.onUserChanged != nil {
		s.onUserChanged(id)
	}

	return user, nil
}

// ChangeUserSettings updates the settings for a user.
func (s *Service) ChangeUserSettings(user *happydns.User, newSettings happydns.UserSettings) error {
	user.Settings = newSettings
	if err := s.store.CreateOrUpdateUser(user); err != nil {
		return err
	}

	if s.onUserChanged != nil {
		s.onUserChanged(user.Id)
	}

	return nil
}

// DeleteUser deletes a user by their identifier.
// This route is for external accounts only.
func (s *Service) DeleteUser(userid happydns.Identifier) error {
	// Disallow route if user is authenticated through local service
	if _, err := s.authUser.GetAuthUser(userid); err == nil {
		return fmt.Errorf("This route is for external account only. Please use the route ./delete instead.")
	}

	if err := s.store.DeleteUser(userid); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteAuthUser in deleteauthuser: %s", err.Error()),
			UserMessage: "Sorry, we are currently unable to delete your profile. Please try again later.",
		}
	}

	return s.closeUserSessions.ByID(userid)
}

// GenerateUserAvatar generates an avatar image for the user.
func (s *Service) GenerateUserAvatar(user *happydns.User, size int, writer io.Writer) error {
	return avatar.GenerateUserAvatar(user, size, writer)
}
