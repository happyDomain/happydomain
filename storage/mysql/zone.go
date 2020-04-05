package database

import (
	"git.happydns.org/happydns/model"
)

func (s *MySQLStorage) GetZones(u *happydns.User) (zones happydns.Zones, err error) {
	if rows, errr := s.db.Query("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE id_user = ?", u.Id); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var z happydns.Zone
			if err = rows.Scan(&z.Id, &z.IdUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility); err != nil {
				return
			}
			zones = append(zones, &z)
		}
		if err = rows.Err(); err != nil {
			return
		}

		return
	}
}

func (s *MySQLStorage) GetZone(u *happydns.User, id int) (z *happydns.Zone, err error) {
	z = &happydns.Zone{}
	err = s.db.QueryRow("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE id_zone=? AND id_user=?", id, u.Id).Scan(&z.Id, &z.IdUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility)
	return
}

func (s *MySQLStorage) GetZoneByDN(u *happydns.User, dn string) (z *happydns.Zone, err error) {
	z = &happydns.Zone{}
	err = s.db.QueryRow("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE domain=? AND id_user=?", dn, u.Id).Scan(&z.Id, &z.IdUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility)
	return
}

func (s *MySQLStorage) ZoneExists(dn string) bool {
	var z int
	err := s.db.QueryRow("SELECT 1 FROM zones WHERE domain=?", dn).Scan(&z)
	return err == nil && z == 1
}

func (s *MySQLStorage) CreateZone(u *happydns.User, z *happydns.Zone) error {
	if res, err := s.db.Exec("INSERT INTO zones (id_user, domain, server, key_name, key_blob, storage_facility) VALUES (?, ?, ?, ?, ?, ?)", u.Id, z.DomainName, z.Server, z.KeyName, z.KeyBlob, z.StorageFacility); err != nil {
		return err
	} else if z.Id, err = res.LastInsertId(); err != nil {
		return err
	} else {
		return nil
	}
}

func (s *MySQLStorage) UpdateZone(z *happydns.Zone) error {
	if _, err := s.db.Exec("UPDATE zones SET domain = ?, key_name = ?, key_algo = ?, key_blob = ?, storage_facility = ? WHERE id_zone = ?", z.DomainName, z.KeyName, z.KeyAlgo, z.KeyBlob, z.StorageFacility, z.Id); err != nil {
		return err
	} else {
		return nil
	}
}

func (s *MySQLStorage) UpdateZoneOwner(z *happydns.Zone, newOwner *happydns.User) error {
	if _, err := s.db.Exec("UPDATE zones SET id_user = ? WHERE id_zone = ?", newOwner.Id, z.Id); err != nil {
		return err
	} else {
		z.IdUser = newOwner.Id
		return nil
	}
}

func (s *MySQLStorage) DeleteZone(z *happydns.Zone) error {
	_, err := s.db.Exec("DELETE FROM zones WHERE id_zone = ?", z.Id)
	return err
}

func (s *MySQLStorage) ClearZones() error {
	_, err := s.db.Exec("DELETE FROM zones")
	return err
}
