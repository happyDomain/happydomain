package database

import (
	"git.happydns.org/happydns/model"
)

func (s *MySQLStorage) GetDomains(u *happydns.User) (domains happydns.Domains, err error) {
	if rows, errr := s.db.Query("SELECT id_domain, id_user, id_source, domain FROM domains WHERE id_user = ?", u.Id); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var z happydns.Domain
			if err = rows.Scan(&z.Id, &z.IdUser, &z.IdSource, &z.DomainName); err != nil {
				return
			}
			domains = append(domains, &z)
		}
		if err = rows.Err(); err != nil {
			return
		}

		return
	}
}

func (s *MySQLStorage) GetDomain(u *happydns.User, id int) (z *happydns.Domain, err error) {
	z = &happydns.Domain{}
	err = s.db.QueryRow("SELECT id_domain, id_user, id_source, domain FROM domains WHERE id_domain=? AND id_user=?", id, u.Id).Scan(&z.Id, &z.IdUser, &z.IdSource, &z.DomainName)
	return
}

func (s *MySQLStorage) GetDomainByDN(u *happydns.User, dn string) (z *happydns.Domain, err error) {
	z = &happydns.Domain{}
	err = s.db.QueryRow("SELECT id_domain, id_user, id_source, domain FROM domains WHERE domain=? AND id_user=?", dn, u.Id).Scan(&z.Id, &z.IdUser, &z.IdSource, &z.DomainName)
	return
}

func (s *MySQLStorage) DomainExists(dn string) bool {
	var z int
	err := s.db.QueryRow("SELECT 1 FROM domains WHERE domain=?", dn).Scan(&z)
	return err == nil && z == 1
}

func (s *MySQLStorage) CreateDomain(u *happydns.User, src happydns.SourceType, z *happydns.Domain) error {
	if res, err := s.db.Exec("INSERT INTO domains (id_user, id_source, domain) VALUES (?, ?, ?)", u.Id, src.Id, z.DomainName); err != nil {
		return err
	} else if z.Id, err = res.LastInsertId(); err != nil {
		return err
	} else {
		return nil
	}
}

func (s *MySQLStorage) UpdateDomain(z *happydns.Domain) error {
	if _, err := s.db.Exec("UPDATE domains SET id_source = ?, domain = ? WHERE id_domain = ?", z.IdSource, z.DomainName, z.Id); err != nil {
		return err
	} else {
		return nil
	}
}

func (s *MySQLStorage) UpdateDomainOwner(z *happydns.Domain, newOwner *happydns.User) error {
	if _, err := s.db.Exec("UPDATE domains SET id_user = ? WHERE id_domain = ?", newOwner.Id, z.Id); err != nil {
		return err
	} else {
		z.IdUser = newOwner.Id
		return nil
	}
}

func (s *MySQLStorage) DeleteDomain(z *happydns.Domain) error {
	_, err := s.db.Exec("DELETE FROM domains WHERE id_domain = ?", z.Id)
	return err
}

func (s *MySQLStorage) ClearDomains() error {
	_, err := s.db.Exec("DELETE FROM domains")
	return err
}
