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

package usecase

import (
	"errors"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

func (tu *tidyUpUsecase) TidyProviders(dropInvalid bool) error {
	iter, err := tu.store.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	err = iterateTidy(iter, dropInvalid, func(prvd *happydns.ProviderMessage) error {
		_, err := tu.store.GetUser(prvd.Owner)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan provider (user %s not found): %v\n", prvd.Owner.String(), prvd)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Iterating drops orphan providers above via DropItem, which removes only
	// the primary key and not the provider-sharing index entries. Reconcile
	// those grants afterwards: a share is orphaned when either the shared
	// provider or the grantee user no longer exists (dropped here, or when a
	// user was deleted, which never touched the share index).
	return tu.tidyProviderShares()
}

// tidyProviderShares drops provider-sharing grants that point to a provider or
// a grantee user that no longer exists.
func (tu *tidyUpUsecase) tidyProviderShares() error {
	bindings, err := tu.store.ListAllProviderShares()
	if err != nil {
		return err
	}

	for _, b := range bindings {
		reason := ""
		if _, err := tu.store.GetProvider(b.ProviderId); errors.Is(err, happydns.ErrProviderNotFound) {
			reason = fmt.Sprintf("provider %s not found", b.ProviderId.String())
		} else if _, err := tu.store.GetUser(b.GranteeId); errors.Is(err, happydns.ErrUserNotFound) {
			reason = fmt.Sprintf("grantee %s not found", b.GranteeId.String())
		}
		if reason == "" {
			continue
		}

		log.Printf("Deleting orphan provider share (%s): provider=%s grantee=%s\n", reason, b.ProviderId.String(), b.GranteeId.String())
		if err := tu.store.DeleteProviderShare(b.ProviderId, b.GranteeId); err != nil {
			return err
		}
	}
	return nil
}
