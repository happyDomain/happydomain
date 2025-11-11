// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/oapi-codegen/runtime/types"

	"git.happydns.org/happyDomain/model"
)

func migrateFrom1(s *LevelDBStorage) (err error) {
	err = migrateFrom1_users_tree(s)
	if err != nil {
		return
	}

	// As session format changed, clear existing
	err = s.ClearSessions()
	if err != nil {
		return
	}

	return
}

type userV1 struct {
	Id               int64
	Email            string
	Password         []byte
	RegistrationTime *time.Time
	EmailValidated   *time.Time
	Settings         happydns.UserSettings
}

func genUserIdv2(input int64) (string, []byte, error) {
	decoded, err := hex.DecodeString(fmt.Sprintf("%x", input))
	return hex.EncodeToString(decoded), decoded, err
}

func migrateFrom1_users_tree(s *LevelDBStorage) (err error) {
	iter := s.search("user-")
	defer iter.Release()

	for iter.Next() {
		var user userV1
		err = decodeData(iter.Value(), &user)
		if err != nil {
			return
		}

		newId, idRaw, errr := genUserIdv2(user.Id)
		if err != nil {
			log.Printf("Migrating v1 -> v2: %s: unable to calculate new ID: %s", iter.Key(), errr.Error())
			continue
		} else if len(idRaw) == 0 {
			log.Printf("Migrating v1 -> v2: %s: unable to calculate new ID: %v", iter.Key(), user)
			continue
		}

		var creationTime time.Time
		if user.RegistrationTime != nil {
			creationTime = *user.RegistrationTime
		} else {
			creationTime = time.Now()
		}

		newUser := &happydns.User{
			Id:        idRaw,
			Email:     types.Email(user.Email),
			CreatedAt: creationTime,
			Settings:  &user.Settings,
		}

		user4auth := &happydns.UserAuth{
			Id:                idRaw,
			Email:             user.Email,
			EmailVerification: user.EmailValidated,
			Password:          user.Password,
			CreatedAt:         creationTime,
			AllowCommercials:  user.Settings.Newsletter,
		}

		log.Printf("Migrating v1 -> v2: %s to user-%x...", iter.Key(), idRaw)

		err = s.put(fmt.Sprintf("user-%x", idRaw), newUser)
		if err != nil {
			return
		}

		err = s.put(fmt.Sprintf("auth-%x", idRaw), user4auth)
		if err != nil {
			return
		}

		err = s.delete(string(iter.Key()))
		if err != nil {
			return
		}

		// Migrate object of the user
		err = errors.Join(
			migrateFrom1_domains(s, user.Id, newId),
			migrateFrom1_provider(s, user.Id, newId),
			migrateFrom1_zone(s, user.Id, newId),
		)
	}

	return
}

func migrateFrom1_domains(s *LevelDBStorage, oldUserId int64, newUserId string) (err error) {
	oldIdStr := []byte(fmt.Sprintf("\"id_owner\":%d", oldUserId))
	newIdStr := []byte(fmt.Sprintf("\"id_owner\":\"%s\"", newUserId))

	iter := s.search("domain-")
	defer iter.Release()

	for iter.Next() {
		domstr := iter.Value()

		migstr := bytes.Replace(domstr, oldIdStr, newIdStr, 1)

		if !bytes.Equal(migstr, domstr) {
			log.Printf("Migrating v1 -> v2: %s...", iter.Key())

			err = s.db.Put(iter.Key(), migstr, nil)
			if err != nil {
				return
			}
		}
	}

	return
}

func migrateFrom1_provider(s *LevelDBStorage, oldUserId int64, newUserId string) (err error) {
	oldIdStr := []byte(fmt.Sprintf("\"_ownerid\":%d", oldUserId))
	newIdStr := []byte(fmt.Sprintf("\"_ownerid\":\"%s\"", newUserId))

	iter := s.search("provider-")
	defer iter.Release()

	for iter.Next() {
		domstr := iter.Value()

		migstr := bytes.Replace(domstr, oldIdStr, newIdStr, 1)

		if !bytes.Equal(migstr, domstr) {
			log.Printf("Migrating v1 -> v2: %s...", iter.Key())

			err = s.db.Put(iter.Key(), migstr, nil)
			if err != nil {
				return
			}
		}
	}

	return
}

func migrateFrom1_zone(s *LevelDBStorage, oldUserId int64, newUserId string) (err error) {
	oldIdStr := []byte(fmt.Sprintf("\"id_author\":%d", oldUserId))
	newIdStr := []byte(fmt.Sprintf("\"id_author\":\"%s\"", newUserId))

	iter := s.search("domain.zone-")
	defer iter.Release()

	for iter.Next() {
		domstr := iter.Value()

		migstr := bytes.Replace(domstr, oldIdStr, newIdStr, 1)

		if !bytes.Equal(migstr, domstr) {
			log.Printf("Migrating v1 -> v2: %s...", iter.Key())

			err = s.db.Put(iter.Key(), migstr, nil)
			if err != nil {
				return
			}
		}
	}

	return
}
