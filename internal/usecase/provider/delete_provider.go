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

package provider

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type DeleteProviderUsecase struct {
	//listDomains *domainUC.ListDomainUsecase
	store ProviderStorage
}

func NewDeleteProviderUsecase(store ProviderStorage) *DeleteProviderUsecase {
	return &DeleteProviderUsecase{
		store: store,
	}
}

func (uc *DeleteProviderUsecase) Delete(user *happydns.User, providerID happydns.Identifier) error {
	// TODO: Find another way to avoid import cycle
	/*domains, err := uc.listDomains.List(user)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("failed to list domains: %w", err),
			UserMessage: "Sorry, we are currently unable to perform this action. Please try again later.",
		}
	}

	for _, d := range domains {
		if d.ProviderId.Equals(providerID) {
			return fmt.Errorf("You cannot delete this provider because it is still used by: %s", d.DomainName)
		}
	}*/

	if err := uc.store.DeleteProvider(providerID); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("failed to delete provider %s: %w", providerID.String(), err),
			UserMessage: "Sorry, we are currently unable to delete your provider. Please try again later.",
		}
	}

	return nil
}
