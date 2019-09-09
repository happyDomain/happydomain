package libredns

import (
)

type Zone struct {
	Id         int64  `json:"id"`
	DomainName string `json:"domain"`
	Server     string `json:"server,omitempty"`
	KeyName    string `json:"keyname,omitempty"`
	KeyBlob    []byte `json:"keyblob,omitempty"`
}

func GetZones() (zones []Zone, err error) {
	if rows, errr := DBQuery("SELECT id_zone, domain, server, key_name, key_blob FROM zones"); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var z Zone
			if err = rows.Scan(&z.Id, &z.DomainName, &z.Server, &z.KeyName, &z.KeyBlob); err != nil {
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
	err = DBQueryRow("SELECT id_user, domain, server, key_name, key_blob FROM zones WHERE id_zone=?", id).Scan(&z.Id, &z.DomainName, &z.Server, &z.KeyName, &z.KeyBlob)
	return
}

func GetZoneByDN(dn string) (z Zone, err error) {
	err = DBQueryRow("SELECT id_zone, domain, server, key_name, key_blob FROM zones WHERE domain=?", dn).Scan(&z.Id, &z.DomainName, &z.Server, &z.KeyName, &z.KeyBlob)
	return
}

func ZoneExists(dn string) bool {
	var z int
	err := DBQueryRow("SELECT 1 FROM zones WHERE domain=?", dn).Scan(&z)
	return err == nil && z == 1
}

func (z *Zone) NewZone() (Zone, error) {
	if res, err := DBExec("INSERT INTO zones (domain, server, key_name, key_blob) VALUES (?, ?, ?, ?)", z.DomainName, z.Server, z.KeyName, z.KeyBlob); err != nil {
		return *z, err
	} else if z.Id, err = res.LastInsertId(); err != nil {
		return *z, err
	} else {
		return *z, nil
	}
}

func (z *Zone) Update() (int64, error) {
	if res, err := DBExec("UPDATE zones SET domain = ? WHERE id_zone = ?", z.DomainName, z.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
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

func ClearZones() (int64, error) {
	if res, err := DBExec("DELETE FROM zones"); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}
