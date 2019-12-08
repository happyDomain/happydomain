package happydns

import (
	"encoding/base64"
)

type Zone struct {
	Id              int64  `json:"id"`
	idUser          int64
	DomainName      string `json:"domain"`
	Server          string `json:"server,omitempty"`
	KeyName         string `json:"keyname,omitempty"`
	KeyAlgo         string `json:"algorithm,omitempty"`
	KeyBlob         []byte `json:"keyblob,omitempty"`
	StorageFacility string `json:"storage_facility,omitempty"`
}

func GetZones() (zones []Zone, err error) {
	if rows, errr := DBQuery("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones"); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var z Zone
			if err = rows.Scan(&z.Id, &z.idUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility); err != nil {
				return
			}
			zones = append(zones, z)
		}
		if err = rows.Err(); err != nil {
			return
		}

		return
	}
}

func (u *User) GetZones() (zones []Zone, err error) {
	if rows, errr := DBQuery("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE id_user = ?", u.Id); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var z Zone
			if err = rows.Scan(&z.Id, &z.idUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility); err != nil {
				return
			}
			zones = append(zones, z)
		}
		if err = rows.Err(); err != nil {
			return
		}

		return
	}
}

func GetZone(id int) (z Zone, err error) {
	err = DBQueryRow("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE id_zone=?", id).Scan(&z.Id, &z.idUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility)
	return
}

func (u *User) GetZone(id int) (z Zone, err error) {
	err = DBQueryRow("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE id_zone=? AND id_user=?", id, u.Id).Scan(&z.Id, &z.idUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility)
	return
}

func GetZoneByDN(dn string) (z Zone, err error) {
	err = DBQueryRow("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE domain=?", dn).Scan(&z.Id, &z.idUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility)
	return
}

func (u *User) GetZoneByDN(dn string) (z Zone, err error) {
	err = DBQueryRow("SELECT id_zone, id_user, domain, server, key_name, key_algo, key_blob, storage_facility FROM zones WHERE domain=? AND id_user=?", dn, u.Id).Scan(&z.Id, &z.idUser, &z.DomainName, &z.Server, &z.KeyName, &z.KeyAlgo, &z.KeyBlob, &z.StorageFacility)
	return
}

func ZoneExists(dn string) bool {
	var z int
	err := DBQueryRow("SELECT 1 FROM zones WHERE domain=?", dn).Scan(&z)
	return err == nil && z == 1
}

func (z *Zone) NewZone(u User) (Zone, error) {
	if res, err := DBExec("INSERT INTO zones (id_user, domain, server, key_name, key_blob, storage_facility) VALUES (?, ?, ?, ?, ?, ?)", u.Id, z.DomainName, z.Server, z.KeyName, z.KeyBlob, z.StorageFacility); err != nil {
		return *z, err
	} else if z.Id, err = res.LastInsertId(); err != nil {
		return *z, err
	} else {
		return *z, nil
	}
}

func (z *Zone) Update() (int64, error) {
	if res, err := DBExec("UPDATE zones SET domain = ?, key_name = ?, key_algo = ?, key_blob = ?, storage_facility = ? WHERE id_zone = ?", z.DomainName, z.KeyName, z.KeyAlgo, z.KeyBlob, z.StorageFacility, z.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}

func (z *Zone) UpdateOwner(u User) (int64, error) {
	if res, err := DBExec("UPDATE zones SET id_user = ? WHERE id_zone = ?", u.Id, z.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		z.idUser = u.Id
		return nb, err
	}
}

func (z *Zone) Delete() (int64, error) {
	if res, err := DBExec("DELETE FROM zones WHERE id_zone = ?", z.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}

func (z *Zone) Base64KeyBlob() string {
	return base64.StdEncoding.EncodeToString(z.KeyBlob)
}

func ClearZones() (int64, error) {
	if res, err := DBExec("DELETE FROM zones"); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}
