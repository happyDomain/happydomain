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
	"log"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type tidyUpUsecase struct {
	store storage.Storage
}

func NewTidyUpUsecase(store storage.Storage) happydns.TidyUpUseCase {
	return &tidyUpUsecase{
		store: store,
	}
}

func (tu *tidyUpUsecase) TidyAll() error {
	for _, tidy := range []func() error{tu.TidySessions, tu.TidyAuthUsers, tu.TidyUsers, tu.TidyProviders, tu.TidyDomains, tu.TidyZones, tu.TidyDomainLogs} {
		if err := tidy(); err != nil {
			return err
		}
	}
	return nil
}

func (tu *tidyUpUsecase) TidyAuthUsers() error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		userAuth := iter.Item()

		_, err = tu.store.GetUser(userAuth.Id)
		if errors.Is(err, storage.ErrNotFound) && time.Since(userAuth.CreatedAt) > 24*time.Hour {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan authuser (user %s not found): %v\n", userAuth.Id.String(), userAuth)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tu *tidyUpUsecase) TidyDomains() error {
	iter, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		domain := iter.Item()

		if _, err = tu.store.GetUser(domain.Owner); err == storage.ErrNotFound {
			// Drop domain of unexistant users
			log.Printf("Deleting orphan domain (user %s not found): %v\n", domain.Owner.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}

		if _, err = tu.store.GetProvider(domain.IdProvider); err == storage.ErrNotFound {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain (provider %s not found): %v\n", domain.IdProvider.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tu *tidyUpUsecase) TidyDomainLogs() error {
	iter, err := tu.store.ListAllDomainLogs()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		l := iter.Item()

		if _, err = tu.store.GetDomain(l.DomainId); err == storage.ErrNotFound {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain log (domain %s not found): %v\n", l.DomainId.String(), l)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tu *tidyUpUsecase) TidyProviders() error {
	iter, err := tu.store.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		prvd := iter.Item()

		_, err = tu.store.GetUser(prvd.Owner)
		if errors.Is(err, storage.ErrNotFound) {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan provider (user %s not found): %v\n", prvd.Owner.String(), prvd)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tu *tidyUpUsecase) TidySessions() error {
	iter, err := tu.store.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		session := iter.Item()

		_, err = tu.store.GetUser(session.IdUser)
		if err == storage.ErrNotFound {
			// Drop session from unexistant users
			log.Printf("Deleting orphan session (user %s not found): %v\n", session.IdUser.String(), session)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (tu *tidyUpUsecase) TidyUsers() error {
	return nil
}

func (tu *tidyUpUsecase) TidyZones() error {
	iterdn, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iterdn.Close()

	var referencedZones []happydns.Identifier

	for iterdn.Next() {
		domain := iterdn.Item()
		for _, zh := range domain.ZoneHistory {
			referencedZones = append(referencedZones, zh)
		}
	}

	iter, err := tu.store.ListAllZones()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		zone := iter.Item()

		foundZone := false
		for _, zid := range referencedZones {
			if zid.Equals(zone.Id) {
				foundZone = true
				break
			}
		}

		if !foundZone {
			// Drop orphan zones
			log.Printf("Deleting orphan zone: %s\n", zone.Id.String())
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return nil
}
