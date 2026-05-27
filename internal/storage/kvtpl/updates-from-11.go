// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package database

import (
	"log"
	"strings"
)

// migrateFrom11 backfills the new secondary indexes introduced alongside
// this migration:
//
//	auth.email|{hash(email)}                 -> {authUserId}
//	domain.owner|{ownerId}|{domainId}        -> ""
//	domain.fqdn|{hash(fqdn)}|{domainId}      -> ""
//	provider.owner|{ownerId}|{providerId}    -> true
//	user.email|{hash(email)}                 -> {userId}
//	user.session-user|{userId}|{sessionHash} -> ""
//
// Before this migration, ListDomains(user), ListProviders(user) and
// FindDomainsByName(fqdn) had to scan every domain or provider in the
// database, GetUserByEmail / GetAuthUserByEmail (plus their *Exists
// counterparts) had to scan every user, and ListAuthUserSessions /
// ListUserSessions had to scan every session; the indexes turn all of
// them into bounded prefix or point lookups.
func migrateFrom11(s *KVStorage) error {
	if err := migrateFrom11_domains(s); err != nil {
		return err
	}
	if err := migrateFrom11_users(s); err != nil {
		return err
	}
	if err := migrateFrom11_authUsers(s); err != nil {
		return err
	}
	if err := migrateFrom11_providers(s); err != nil {
		return err
	}
	return migrateFrom11_sessions(s)
}

func migrateFrom11_domains(s *KVStorage) error {
	iter, err := s.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		d := iter.Item()
		if d == nil {
			continue
		}
		if err := s.putDomainIndexes(d); err != nil {
			return err
		}
		n++
	}

	if err := iter.Err(); err != nil {
		return err
	}

	log.Printf("migrateFrom11: backfilled owner+fqdn indexes for %d domains", n)
	return nil
}

func migrateFrom11_users(s *KVStorage) error {
	iter, err := s.ListAllUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		u := iter.Item()
		if u == nil || u.Email == "" {
			continue
		}
		if err := s.db.Put(userEmailIndexKey(u.Email), u.Id.String()); err != nil {
			return err
		}
		n++
	}

	if err := iter.Err(); err != nil {
		return err
	}

	log.Printf("migrateFrom11: backfilled user-email index for %d users", n)
	return nil
}

func migrateFrom11_authUsers(s *KVStorage) error {
	iter, err := s.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		u := iter.Item()
		if u == nil || u.Email == "" {
			continue
		}
		if err := s.db.Put(authEmailIndexKey(u.Email), u.Id.String()); err != nil {
			return err
		}
		n++
	}

	if err := iter.Err(); err != nil {
		return err
	}

	log.Printf("migrateFrom11: backfilled auth-email index for %d auth users", n)
	return nil
}

func migrateFrom11_providers(s *KVStorage) error {
	iter, err := s.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		p := iter.Item()
		if p == nil {
			continue
		}
		if err := s.db.Put(providerOwnerKey(p.Owner, p.Id), true); err != nil {
			return err
		}
		n++
	}

	if err := iter.Err(); err != nil {
		return err
	}

	log.Printf("migrateFrom11: backfilled provider-owner index for %d providers", n)
	return nil
}

func migrateFrom11_sessions(s *KVStorage) error {
	iter, err := s.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		session := iter.Item()
		if session == nil || session.IdUser.IsEmpty() {
			continue
		}

		// The primary key embeds the session hash as its suffix after
		// the "user.session-" prefix; reuse it directly so we don't
		// have to re-derive it from the (already discarded) raw id.
		hash := strings.TrimPrefix(iter.Key(), sessionPrimaryPrefix)
		if hash == "" || hash == iter.Key() {
			log.Printf("migrateFrom11: skipping session with unexpected key %q", iter.Key())
			continue
		}

		if err := s.db.Put(sessionUserIndexKey(session.IdUser, hash), ""); err != nil {
			return err
		}
		n++
	}

	if err := iter.Err(); err != nil {
		return err
	}

	log.Printf("migrateFrom11: backfilled session user index for %d sessions", n)
	return nil
}
