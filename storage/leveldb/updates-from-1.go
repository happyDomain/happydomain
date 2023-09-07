// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package database

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"time"

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

	err = s.Tidy()
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
			Email:     user.Email,
			CreatedAt: creationTime,
			Settings:  user.Settings,
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
		migrateFrom1_domains(s, user.Id, newId)
		migrateFrom1_provider(s, user.Id, newId)
		migrateFrom1_zone(s, user.Id, newId)
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
