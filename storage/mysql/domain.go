// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

func (s *MySQLStorage) GetDomain(u *happydns.User, id int64) (z *happydns.Domain, err error) {
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
