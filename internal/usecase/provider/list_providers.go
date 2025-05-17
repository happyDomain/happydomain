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

type ListProvidersUsecase struct {
	store ProviderStorage
}

func NewListProvidersUsecase(store ProviderStorage) *ListProvidersUsecase {
	return &ListProvidersUsecase{
		store: store,
	}
}

func (uc *ListProvidersUsecase) List(user *happydns.User) ([]*happydns.ProviderMeta, error) {
	items, err := uc.store.ListProviders(user)
	if err != nil {
		return nil, fmt.Errorf("list providers failed: %w", err)
	}

	metas := make([]*happydns.ProviderMeta, 0, len(items))
	for _, p := range items {
		metas = append(metas, &p.ProviderMeta)
	}

	return metas, nil
}
