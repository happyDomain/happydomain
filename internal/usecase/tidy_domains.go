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

func (tu *tidyUpUsecase) TidyDomains(dropInvalid bool) error {
	iter, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	err = iterateTidy(iter, dropInvalid, func(domain *happydns.Domain) error {
		if _, err := tu.store.GetUser(domain.Owner); errors.Is(err, happydns.ErrUserNotFound) {
			// Drop domain of unexistant users
			log.Printf("Deleting orphan domain (user %s not found): %v\n", domain.Owner.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}

		if _, err := tu.store.GetProvider(domain.ProviderId); errors.Is(err, happydns.ErrProviderNotFound) {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain (provider %s not found): %v\n", domain.ProviderId.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := tu.store.TidyDomainIndexes(); err != nil {
		return err
	}

	// Iterating drops orphan domains above via DropItem, which removes only the
	// primary key and not the domain-sharing index entries. Reconcile those
	// grants afterwards: a share is orphaned when either the shared domain or
	// the grantee user no longer exists (dropped here, or when a user was
	// deleted, which never touched the share indexes).
	return tu.tidyDomainShares()
}

// tidyDomainShares drops domain-sharing grants that point to a domain or a
// grantee user that no longer exists. Deleting a share removes both the share
// and grant index entries.
func (tu *tidyUpUsecase) tidyDomainShares() error {
	bindings, err := tu.store.ListAllDomainShares()
	if err != nil {
		return err
	}

	for _, b := range bindings {
		reason := ""
		if _, err := tu.store.GetDomain(b.DomainId); errors.Is(err, happydns.ErrDomainNotFound) {
			reason = fmt.Sprintf("domain %s not found", b.DomainId.String())
		} else if _, err := tu.store.GetUser(b.GranteeId); errors.Is(err, happydns.ErrUserNotFound) {
			reason = fmt.Sprintf("grantee %s not found", b.GranteeId.String())
		}
		if reason == "" {
			continue
		}

		log.Printf("Deleting orphan domain share (%s): domain=%s grantee=%s\n", reason, b.DomainId.String(), b.GranteeId.String())
		if err := tu.store.DeleteDomainShare(b.DomainId, b.GranteeId); err != nil {
			return err
		}
	}
	return nil
}

func (tu *tidyUpUsecase) TidyDomainLogs(dropInvalid bool) error {
	iter, err := tu.store.ListAllDomainLogs()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(l *happydns.DomainLogWithDomainId) error {
		if _, err := tu.store.GetDomain(l.DomainId); errors.Is(err, happydns.ErrDomainNotFound) {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain log (domain %s not found): %v\n", l.DomainId.String(), l)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}
