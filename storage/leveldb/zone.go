package database

import (
	"fmt"

	"git.happydns.org/happydns/model"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetZones(u *happydns.User) (zones happydns.Zones, err error) {
	iter := s.search("zone-")
	defer iter.Release()

	for iter.Next() {
		var z happydns.Zone

		err = decodeData(iter.Value(), &z)
		if err != nil {
			return
		}

		if z.IdUser == u.Id {
			zones = append(zones, &z)
		}
	}

	return
}

func (s *LevelDBStorage) GetZone(u *happydns.User, id int) (z *happydns.Zone, err error) {
	z = &happydns.Zone{}
	err = s.get(fmt.Sprintf("zone-%d", id), &z)

	if z.IdUser != u.Id {
		z = nil
		err = leveldb.ErrNotFound
	}

	return
}

func (s *LevelDBStorage) GetZoneByDN(u *happydns.User, dn string) (*happydns.Zone, error) {
	zones, err := s.GetZones(u)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if zone.DomainName == dn {
			return zone, nil
		}
	}

	return nil, leveldb.ErrNotFound
}

func (s *LevelDBStorage) ZoneExists(dn string) bool {
	iter := s.search("zone-")
	defer iter.Release()

	for iter.Next() {
		var z happydns.Zone

		err := decodeData(iter.Value(), &z)
		if err != nil {
			continue
		}

		if z.DomainName == dn {
			return true
		}
	}

	return false
}

func (s *LevelDBStorage) CreateZone(u *happydns.User, z *happydns.Zone) error {
	key, id, err := s.findInt63Key("zone-")
	if err != nil {
		return err
	}

	z.Id = id
	z.IdUser = u.Id
	return s.put(key, z)
}

func (s *LevelDBStorage) UpdateZone(z *happydns.Zone) error {
	return s.put(fmt.Sprintf("zone-%d", z.Id), z)
}

func (s *LevelDBStorage) UpdateZoneOwner(z *happydns.Zone, newOwner *happydns.User) error {
	z.IdUser = newOwner.Id
	return s.put(fmt.Sprintf("zone-%d", z.Id), z)
}

func (s *LevelDBStorage) DeleteZone(z *happydns.Zone) error {
	return s.delete(fmt.Sprintf("zone-%d", z.Id))
}

func (s *LevelDBStorage) ClearZones() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("zone-")), nil)
	defer iter.Release()

	for iter.Next() {
		err = tx.Delete(iter.Key(), nil)
		if err != nil {
			tx.Discard()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Discard()
		return err
	}

	return nil
}
