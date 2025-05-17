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
	"git.happydns.org/happyDomain/model"
)

type GetProviderUsecase struct {
	store ProviderStorage
}

func NewGetProviderUsecase(store ProviderStorage) *GetProviderUsecase {
	return &GetProviderUsecase{
		store: store,
	}
}

func (uc *GetProviderUsecase) getUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMessage, error) {
	p, err := uc.store.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(p.ProviderMeta.Owner) {
		return nil, happydns.ErrProviderNotFound
	}

	return p, err
}

func (uc *GetProviderUsecase) Get(user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	p, err := uc.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
	}

	return ParseProvider(p)
}

func (uc *GetProviderUsecase) GetMeta(user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMeta, error) {
	p, err := uc.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
	}

	return p.Meta(), nil
}
