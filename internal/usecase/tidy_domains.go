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

	return tu.store.TidyDomainIndexes()
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

// TidyDomainWatches deletes the domain watches whose user no longer exists,
// then the user index entries left pointing to a missing user or watch.
func (tu *tidyUpUsecase) TidyDomainWatches(dropInvalid bool) error {
	iter, err := tu.store.ListAllDomainWatches()
	if err != nil {
		return err
	}

	err = tidyUserOrphans(tu, iter, dropInvalid,
		func(w *happydns.DomainWatch) happydns.Identifier { return w.UserId },
		func(w *happydns.DomainWatch) string { return "domain watch " + w.Id.String() + " (" + w.DomainName + ")" },
		func(w *happydns.DomainWatch) error { return tu.store.DeleteDomainWatch(w.Id) },
		happydns.ErrDomainWatchNotFound,
	)
	if err != nil {
		return err
	}

	return tu.store.TidyDomainWatchIndexes()
}

// TidyZoneTemplates deletes the zone templates whose user no longer exists,
// then the index entries left pointing to a missing user or template. The
// global templates have no user: they are never orphans.
func (tu *tidyUpUsecase) TidyZoneTemplates(dropInvalid bool) error {
	iter, err := tu.store.ListAllZoneTemplates()
	if err != nil {
		return err
	}

	// Not tidyUserOrphans: a global template has no user to look up.
	var orphans []*happydns.ZoneTemplateMessage
	err = iterateTidy(iter, dropInvalid, func(t *happydns.ZoneTemplateMessage) error {
		if t.IsGlobal() {
			return nil
		}
		_, err := tu.store.GetUser(*t.UserId)
		if errors.Is(err, happydns.ErrUserNotFound) {
			orphans = append(orphans, t)
			return nil
		}
		return err
	})
	iter.Close()
	if err != nil {
		return err
	}

	for _, t := range orphans {
		log.Printf("Deleting orphan zone template %s (%s): user not found", t.Id.String(), t.Name)
		if err := tu.store.DeleteZoneTemplate(t.Id); err != nil && !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
			return fmt.Errorf("unable to delete orphan zone template %s: %w", t.Id.String(), err)
		}
	}

	return tu.store.TidyZoneTemplateIndexes()
}
