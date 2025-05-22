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
	"testing"

	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

func TestCanRegisterUsecase_IsOpened(t *testing.T) {
	// Test case where registration is enabled
	t.Run("Registration is enabled", func(t *testing.T) {
		cfg := &happydns.Options{DisableRegistration: false}
		uc := authuser.NewCanRegisterUsecase(cfg)

		if !uc.IsOpened() {
			t.Errorf("Expected registration to be enabled")
		}
	})

	// Test case where registration is disabled
	t.Run("Registration is disabled", func(t *testing.T) {
		cfg := &happydns.Options{DisableRegistration: true}
		uc := authuser.NewCanRegisterUsecase(cfg)

		if uc.IsOpened() {
			t.Errorf("Expected registration to be disabled")
		}
	})
}
