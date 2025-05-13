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

package authuser

import (
	"fmt"

	"git.happydns.org/happyDomain/internal/mailer"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

// Service groups all use cases related to user authentication and management.
type Service struct {
	// Usecases for user management actions
	CanRegisterUC              *CanRegisterUsecase
	ChangePasswordUC           *ChangePasswordUsecase
	CheckPasswordConstraintsUC *CheckPasswordConstraintsUsecase
	CreateAuthUserUC           *CreateAuthUserUsecase
	DeleteAuthUserUC           *DeleteAuthUserUsecase
	EmailValidationUC          *EmailValidationUsecase
	GetAuthUserUC              *GetAuthUserUsecase
	RecoverAccountUC           *RecoverAccountUsecase
}

// NewAuthUserService initializes and returns a new AuthUserService, containing all use cases.
func NewAuthUserUsecases(
	cfg *happydns.Options,
	mailer *mailer.Mailer,
	store AuthUserStorage,
	closeUserSessionsUseCase *sessionUC.CloseUserSessionsUsecase,
) *Service {
	checkPasswordConstraintsUC := NewCheckPasswordConstraintsUsecase()
	changePasswordUC := NewChangePasswordUsecase(store, checkPasswordConstraintsUC)
	emailValidationUC := NewEmailValidationUsecase(store, mailer, cfg)
	getAuthUserUC := NewGetAuthUserUsecase(store)

	// Initialize each usecase by injecting required dependencies.
	return &Service{
		CanRegisterUC:              NewCanRegisterUsecase(cfg),
		ChangePasswordUC:           changePasswordUC,
		CheckPasswordConstraintsUC: checkPasswordConstraintsUC,
		CreateAuthUserUC:           NewCreateAuthUserUsecase(store, mailer, checkPasswordConstraintsUC, emailValidationUC),
		DeleteAuthUserUC:           NewDeleteAuthUserUsecase(store, closeUserSessionsUseCase),
		EmailValidationUC:          emailValidationUC,
		GetAuthUserUC:              getAuthUserUC,
		RecoverAccountUC:           NewRecoverAccountUsecase(store, mailer, cfg, changePasswordUC),
	}
}

func (s *Service) CanRegister(user happydns.UserRegistration) error {
	if !s.CanRegisterUC.IsOpened() {
		return fmt.Errorf("Registration are closed on this instance.")
	}

	return nil
}

func (s *Service) CheckPassword(user *happydns.UserAuth, request happydns.ChangePasswordForm) error {
	return s.ChangePasswordUC.CheckResetPassword(user, request)
}

func (s *Service) CheckNewPassword(user *happydns.UserAuth, request happydns.ChangePasswordForm) error {
	return s.ChangePasswordUC.CheckNewPassword(user, request)
}

func (s *Service) ChangePassword(user *happydns.UserAuth, newPassword string) error {
	return s.ChangePasswordUC.Change(user, newPassword)
}

func (s *Service) CreateAuthUser(uu happydns.UserRegistration) (*happydns.UserAuth, error) {
	return s.CreateAuthUserUC.Create(uu)
}

func (s *Service) DeleteAuthUser(user *happydns.UserAuth, password string) error {
	return s.DeleteAuthUserUC.Delete(user, password)
}

func (s *Service) GetAuthUser(userID happydns.Identifier) (*happydns.UserAuth, error) {
	return s.GetAuthUserUC.ByID(userID)
}

func (s *Service) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	return s.GetAuthUserUC.ByEmail(email)
}

func (s *Service) GenerateRecoveryLink(user *happydns.UserAuth) (string, error) {
	return s.RecoverAccountUC.GenerateLink(user)
}

func (s *Service) SendRecoveryLink(user *happydns.UserAuth) error {
	return s.RecoverAccountUC.SendLink(user)
}

func (s *Service) GenerateValidationLink(user *happydns.UserAuth) string {
	return s.EmailValidationUC.GenerateLink(user)
}

func (s *Service) ResetPassword(user *happydns.UserAuth, form happydns.AccountRecoveryForm) error {
	return s.RecoverAccountUC.ResetPassword(user, form)
}

func (s *Service) SendValidationLink(user *happydns.UserAuth) error {
	return s.EmailValidationUC.SendLink(user)
}

func (s *Service) ValidateEmail(user *happydns.UserAuth, form happydns.AddressValidationForm) error {
	return s.EmailValidationUC.Validate(user, form)
}
