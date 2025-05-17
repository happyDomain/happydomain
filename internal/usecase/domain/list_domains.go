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

package domain

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type ListDomainsUsecase struct {
	store DomainStorage
}

func NewListDomainsUsecase(store DomainStorage) *ListDomainsUsecase {
	return &ListDomainsUsecase{
		store: store,
	}
}

func (uc *ListDomainsUsecase) List(user *happydns.User) ([]*happydns.Domain, error) {
	domains, err := uc.store.ListDomains(user)
	if err != nil {
		return nil, fmt.Errorf("an error occurs when trying to GetUserDomains: %s", err.Error())
	}

	if len(domains) == 0 {
		return []*happydns.Domain{}, nil
	}

	return domains, nil
}
