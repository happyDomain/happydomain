// Copyright or © or Copr. happyDNS (2023)
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
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"git.happydns.org/happydomain/model"

	"github.com/syndtr/goleveldb/leveldb/errors"
)

func migrateFrom2(s *LevelDBStorage) (err error) {
	err = migrateFrom2_users_tree(s)
	if err != nil {
		return
	}

	err = s.Tidy()
	if err != nil {
		return
	}

	return
}

type userV2 struct {
	Id        happydns.HexaString
	Email     string
	CreatedAt time.Time
	LastSeen  time.Time
	Settings  happydns.UserSettings
}

func migrateFrom2_users_tree(s *LevelDBStorage) (err error) {
	iter := s.search("user-")
	defer iter.Release()

	for iter.Next() {
		var user userV2
		err = decodeData(iter.Value(), &user)
		if err != nil {
			return
		}

		newId := happydns.Identifier(user.Id)
		if len(newId) < happydns.IDENTIFIER_LEN {
			newId, err = happydns.NewRandomIdentifier()
			if err != nil {
				return fmt.Errorf("unable to generate a new identifier for %s: %w", iter.Key(), err)
			}
		}

		newUser := &happydns.User{
			Id:        newId,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			LastSeen:  user.LastSeen,
			Settings:  user.Settings,
		}

		log.Printf("Migrating v2 -> v3: %s to user-%s...", iter.Key(), newId.String())

		err = s.put(fmt.Sprintf("user-%s", newId.String()), newUser)
		if err != nil {
			return fmt.Errorf("unable to write %s: %w", iter.Key(), err)
		}

		err = s.delete(string(iter.Key()))
		if err != nil {
			return fmt.Errorf("unable to delete migrated %s: %w", iter.Key(), err)
		}

		// Migrate object of the user
		err = migrateFrom2_auth(s, user.Id, newId, user)
		if err != nil {
			return fmt.Errorf("unable to migrate auth user for user-%s (%s): %w", newId.String(), user.Email, err)
		}

		err = migrateFrom2_session(s, user.Id, newId.String())
		if err != nil {
			return fmt.Errorf("unable to migrate session for user-%s (%s): %w", newId.String(), user.Email, err)
		}

		err = migrateFrom2_provider(s, user.Id, newId.String())
		if err != nil {
			return fmt.Errorf("unable to migrate providers for user-%s: %w", newId.String(), err)
		}
	}

	return
}

func migrateFrom2_auth(s *LevelDBStorage, oldUserId happydns.HexaString, newId happydns.Identifier, user userV2) (err error) {
	oldIdStr := []byte(fmt.Sprintf("\"Id\":\"%s\"", base64.StdEncoding.EncodeToString(oldUserId)))
	newIdStr := []byte(fmt.Sprintf("\"Id\":\"%s\"", newId.String()))

	oldAuthKey := fmt.Sprintf("auth-%x", oldUserId)

	usrstr, err := s.db.Get([]byte(oldAuthKey), nil)
	if err != nil {
		if err == errors.ErrNotFound {
			user4auth := &happydns.UserAuth{
				Id:                newId,
				Email:             user.Email,
				EmailVerification: nil,
				Password:          nil,
				CreatedAt:         time.Now(),
				AllowCommercials:  false,
			}

			log.Printf("Migrating v2 -> v3: auth-%s: %s not found, creating it", newId.String(), oldAuthKey)

			return s.put(fmt.Sprintf("auth-%s", newId.String()), user4auth)
		}
		return fmt.Errorf("unable to find/decode %s: %w", oldAuthKey, err)
	}

	migstr := bytes.Replace(usrstr, oldIdStr, newIdStr, 1)

	if !bytes.Equal(migstr, usrstr) {
		var newauth happydns.UserAuth
		err = decodeData(migstr, &newauth)
		if err != nil {
			log.Printf("From %s to %s", usrstr, migstr)
			return fmt.Errorf("unable to reconstruct a valid auth user: %w", err)
		}

		err = s.db.Put([]byte(fmt.Sprintf("auth-%s", newId.String())), migstr, nil)
		if err != nil {
			return fmt.Errorf("unable to write auth-%s (from %s): %w", newId.String(), oldAuthKey, err)
		}
		log.Printf("Migrating v2 -> v3: %s to auth-%s...", oldAuthKey, newId.String())

		err = s.delete(oldAuthKey)
		if err != nil {
			return fmt.Errorf("unable to delete migrated %s: %w", oldAuthKey, err)
		}
	}

	return
}

type sessionV2 struct {
	Id []byte `json:"id"`
}

func migrateFrom2_session(s *LevelDBStorage, oldUserId happydns.HexaString, newUserId string) (err error) {
	oldOwnerIdStr := []byte(fmt.Sprintf("\"login\":\"%x\"", oldUserId))
	newOwnerIdStr := []byte(fmt.Sprintf("\"login\":\"%s\"", newUserId))

	iter := s.search("user.session-")
	defer iter.Release()

	for iter.Next() {
		usrstr := iter.Value()

		if bytes.Contains(usrstr, oldOwnerIdStr) {
			var session sessionV2
			err = decodeData(usrstr, &session)
			if err != nil {
				return fmt.Errorf("unable to decode %s: %w", iter.Key(), err)
			}

			newId := happydns.Identifier(session.Id)

			oldIdStr := []byte(fmt.Sprintf("\"id\":\"%s\"", base64.StdEncoding.EncodeToString(session.Id)))
			newIdStr := []byte(fmt.Sprintf("\"id\":\"%s\"", newId.String()))

			migstr := bytes.Replace(usrstr, oldIdStr, newIdStr, 1)
			migstr = bytes.Replace(migstr, oldOwnerIdStr, newOwnerIdStr, 1)

			if !bytes.Equal(migstr, usrstr) {
				err = s.db.Put([]byte(fmt.Sprintf("user.session-%s", newUserId)), migstr, nil)
				if err != nil {
					return fmt.Errorf("unable to write user.session-%s (from %s): %w", newId.String(), iter.Key(), err)
				}
				log.Printf("Migrating v2 -> v3: %s to user.session-%s...", iter.Key(), newId.String())

				err = s.delete(string(iter.Key()))
				if err != nil {
					return fmt.Errorf("unable to delete migrated %s: %w", iter.Key(), err)
				}
			}
		}
	}

	return
}

type providerV2 struct {
	Id      int64               `json:"_id"`
	OwnerId happydns.HexaString `json:"_ownerid"`
}

func migrateFrom2_provider(s *LevelDBStorage, oldUserId happydns.HexaString, newUserId string) (err error) {
	oldOwnerIdStr := []byte(fmt.Sprintf("\"_ownerid\":\"%x\"", oldUserId))
	newOwnerIdStr := []byte(fmt.Sprintf("\"_ownerid\":\"%s\"", newUserId))

	iter := s.search("provider-")
	defer iter.Release()

	for iter.Next() {
		domstr := iter.Value()

		if bytes.Contains(domstr, oldOwnerIdStr) {
			var provider providerV2
			err = decodeData(domstr, &provider)
			if err != nil {
				return fmt.Errorf("unable to decode %s: %w", iter.Key(), err)
			}

			var newId happydns.Identifier
			newId, err = happydns.NewRandomIdentifier()
			if err != nil {
				return fmt.Errorf("unable to generate a new identifier for %s: %w", iter.Key(), err)
			}

			oldIdStr := []byte(fmt.Sprintf("\"_id\":%d", provider.Id))
			newIdStr := []byte(fmt.Sprintf("\"_id\":\"%s\"", newId.String()))

			migstr := bytes.Replace(domstr, oldIdStr, newIdStr, 1)
			migstr = bytes.Replace(migstr, oldOwnerIdStr, newOwnerIdStr, 1)

			if !bytes.Equal(migstr, domstr) {
				var newprv happydns.ProviderMeta
				err = decodeData(migstr, &newprv)
				if err != nil {
					log.Printf("From %s to %s", domstr, migstr)
					return fmt.Errorf("unable to reconstruct a valid provider: %w", err)
				}

				log.Printf("Migrating v2 -> v3: %s...", iter.Key())

				err = s.db.Put([]byte(fmt.Sprintf("provider-%s", newId.String())), migstr, nil)
				if err != nil {
					return
				}

				err = s.delete(string(iter.Key()))
				if err != nil {
					return
				}

				err = migrateFrom2_domains(s, oldUserId, newUserId, provider.Id, newId.String())
				if err != nil {
					return fmt.Errorf("unable to migrate domains for provider-%s: %w", newId.String(), err)
				}
			}
		}
	}

	return
}

type domainV2 struct {
	Id          int64   `json:"id"`
	IdProvider  int64   `json:"id_provider"`
	ZoneHistory []int64 `json:"zone_history"`
}

func migrateFrom2_domains(s *LevelDBStorage, oldUserId happydns.HexaString, newUserId string, oldProviderId int64, newProviderId string) (err error) {
	oldProviderIdStr := []byte(fmt.Sprintf("\"id_provider\":%d", oldProviderId))
	newProviderIdStr := []byte(fmt.Sprintf("\"id_provider\":\"%s\"", newProviderId))
	oldOwnerIdStr := []byte(fmt.Sprintf("\"id_owner\":\"%x\"", oldUserId))
	newOwnerIdStr := []byte(fmt.Sprintf("\"id_owner\":\"%s\"", newUserId))

	iter := s.search("domain-")
	defer iter.Release()

	for iter.Next() {
		domstr := iter.Value()

		if bytes.Contains(domstr, oldProviderIdStr) && bytes.Contains(domstr, oldOwnerIdStr) {
			var domain domainV2
			err = decodeData(domstr, &domain)
			if err != nil {
				return fmt.Errorf("unable to decode %s: %w", iter.Key(), err)
			}

			var newId happydns.Identifier
			newId, err = happydns.NewRandomIdentifier()
			if err != nil {
				return fmt.Errorf("unable to generate a new identifier for %s: %w", iter.Key(), err)
			}

			oldIdStr := []byte(fmt.Sprintf("\"id\":%d", domain.Id))
			newIdStr := []byte(fmt.Sprintf("\"id\":\"%s\"", newId.String()))

			zoneOldStr := []byte("\"zone_history\":[")
			zoneNewStr := []byte("\"zone_history\":[")
			// Migrate zones
			for _, zoneid := range domain.ZoneHistory {
				var newZoneId happydns.Identifier
				newZoneId, err = happydns.NewRandomIdentifier()
				if err != nil {
					return fmt.Errorf("unable to generate a new identifier for a zone of %s: %w", iter.Key(), err)
				}

				err = migrateFrom2_zone(s, oldUserId, newUserId, zoneid, newZoneId.String())
				if err != nil {
					return fmt.Errorf("unable to migrate domain.zone-%d: %w", zoneid, err)
				}

				zoneOldStr = append(zoneOldStr, []byte(fmt.Sprintf("%d,", zoneid))...)
				zoneNewStr = append(zoneNewStr, []byte(fmt.Sprintf("\"%s\",", newZoneId.String()))...)
			}
			zoneOldStr[len(zoneOldStr)-1] = ']'
			zoneNewStr[len(zoneNewStr)-1] = ']'

			migstr := bytes.Replace(domstr, oldIdStr, newIdStr, 1)
			migstr = bytes.Replace(migstr, oldOwnerIdStr, newOwnerIdStr, 1)
			migstr = bytes.Replace(migstr, oldProviderIdStr, newProviderIdStr, 1)
			migstr = bytes.Replace(migstr, zoneOldStr, zoneNewStr, 1)

			if !bytes.Equal(migstr, domstr) {
				var newdn happydns.Domain
				err = decodeData(migstr, &newdn)
				if err != nil {
					log.Printf("From %s to %s", domstr, migstr)
					return fmt.Errorf("unable to reconstruct a valid domain: %w", err)
				}

				log.Printf("Migrating v2 -> v3: %s...", iter.Key())

				err = s.db.Put([]byte(fmt.Sprintf("domain-%s", newId.String())), migstr, nil)
				if err != nil {
					return
				}

				err = s.delete(string(iter.Key()))
				if err != nil {
					return
				}
			}
		}
	}

	return
}

func migrateFrom2_zone(s *LevelDBStorage, oldUserId happydns.HexaString, newUserId string, oldZoneId int64, newZoneId string) (err error) {
	oldIdStr := []byte(fmt.Sprintf("\"id\":%d", oldZoneId))
	newIdStr := []byte(fmt.Sprintf("\"id\":\"%s\"", newZoneId))
	oldIdOwnerStr := []byte(fmt.Sprintf("\"id_author\":%d", oldUserId))
	newIdOwnerStr := []byte(fmt.Sprintf("\"id_author\":\"%s\"", newUserId))

	oldZoneKey := fmt.Sprintf("domain.zone-%d", oldZoneId)

	zonestr, err := s.db.Get([]byte(oldZoneKey), nil)
	if err != nil {
		return fmt.Errorf("unable to find/decode %s: %w", oldZoneKey, err)
	}

	migstr := bytes.Replace(zonestr, oldIdStr, newIdStr, 1)
	migstr = bytes.Replace(migstr, oldIdOwnerStr, newIdOwnerStr, 1)

	if !bytes.Equal(migstr, zonestr) {
		err = s.db.Put([]byte(fmt.Sprintf("domain.zone-%s", newZoneId)), migstr, nil)
		if err != nil {
			return fmt.Errorf("unable to write domain.zone-%s (from %s): %w", newZoneId, oldZoneKey, err)
		}
		log.Printf("Migrating v2 -> v3: %s to domain.zone-%s...", oldZoneKey, newZoneId)

		err = s.delete(oldZoneKey)
		if err != nil {
			return fmt.Errorf("unable to delete migrated %s: %w", oldZoneKey, err)
		}
	}

	return
}
